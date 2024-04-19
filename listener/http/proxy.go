package http

import (
	"fmt"
	"github.com/zhaofenghao/clash/params"
	"net"
	"net/http"
	"strings"

	"github.com/zhaofenghao/clash/adapter/inbound"
	N "github.com/zhaofenghao/clash/common/net"
	"github.com/zhaofenghao/clash/component/auth"
	C "github.com/zhaofenghao/clash/constant"
	"github.com/zhaofenghao/clash/log"
)

//const HttpMessageBytes = `HTTP/1.1 %d %s
//Content-Type: text/plain; charset=utf-8
//Proxy-Authenticate: Basic realm="%s"
//
//errorMsg: %s`

func HandleConn(c net.Conn, in chan<- C.ConnContext, authenticator auth.Authenticator) {
	client := newClient(c, in)
	defer client.CloseIdleConnections()

	conn := N.NewBufferedConn(c)

	keepAlive := true
	trusted := false // disable authenticate if cache is nil

	for keepAlive {
		request, err := ReadRequest(conn.Reader())
		if err != nil {
			break
		}

		request.RemoteAddr = conn.RemoteAddr().String()

		keepAlive = strings.TrimSpace(strings.ToLower(request.Header.Get("Proxy-Connection"))) == "keep-alive"

		var resp *http.Response
		var ctx = new(params.ValueContext)
		ctx.SetRemoteIP(c.RemoteAddr().String())

		if !trusted {
			err, resp = authenticate(ctx, request, authenticator)
			//if err != nil {
			//	fmt.Fprintf(c, "%s", err.Error())
			//	conn.Close()
			//	return
			//}
			request = request.WithContext(ctx.GetContext())
			trusted = resp == nil
		}

		if trusted {
			if request.Method == http.MethodConnect {
				// Manual writing to support CONNECT for http 1.0 (workaround for uplay client)
				if _, err = fmt.Fprintf(conn, "HTTP/%d.%d %03d %s\r\n\r\n", request.ProtoMajor, request.ProtoMinor, http.StatusOK, "Connection established"); err != nil {
					break // close connection
				}

				in <- inbound.NewHTTPS(request, conn)

				return // hijack connection
			}

			host := request.Header.Get("Host")
			if host != "" {
				request.Host = host
			}

			request.RequestURI = ""

			if isUpgradeRequest(request) {
				handleUpgrade(conn, request, in)

				return // hijack connection
			}

			removeHopByHopHeaders(request.Header)
			removeExtraHTTPHostPort(request)

			if request.URL.Scheme == "" || request.URL.Host == "" {
				resp = responseWith(request, http.StatusBadRequest)
			} else {
				resp, err = client.Do(request)
				if err != nil {
					resp = responseWith(request, http.StatusBadGateway)
				}
			}

			removeHopByHopHeaders(resp.Header)
		}

		if keepAlive {
			resp.Header.Set("Proxy-Connection", "keep-alive")
			resp.Header.Set("Connection", "keep-alive")
			resp.Header.Set("Keep-Alive", "timeout=4")
		}

		resp.Close = !keepAlive

		err = resp.Write(conn)
		if err != nil {
			fmt.Fprintf(c, "%s", err.Error())
			break // close connection
		}
	}

	conn.Close()
}

func authenticate(ctx *params.ValueContext, request *http.Request, authenticator auth.Authenticator) (error, *http.Response) {
	var err error
	if authenticator != nil {
		credential := parseBasicProxyAuthorization(request)
		var authed = false
		if credential == "" {
			var isSupport bool
			authed, isSupport = authenticator.VerifyByIp(ctx)
			if !isSupport {
				resp := responseWith(request, http.StatusProxyAuthRequired)
				resp.Header.Set("Proxy-Authenticate", "Basic")
				return nil, resp
			}
		} else {
			user, pass, err := decodeBasicProxyAuthorization(credential)
			authed = err == nil && authenticator.Verify(ctx, user, pass)
		}
		if !authed {
			log.Infoln("Auth failed from %s !!", request.RemoteAddr)
			err = fmt.Errorf("auth failed from %s", request.RemoteAddr)
			return err, responseWith(request, http.StatusForbidden)
		}
	}

	return err, nil
}

func responseWith(request *http.Request, statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Proto:      request.Proto,
		ProtoMajor: request.ProtoMajor,
		ProtoMinor: request.ProtoMinor,
		Header:     http.Header{},
	}
}

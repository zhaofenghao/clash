package mixed

import (
	"context"
	"github.com/Dreamacro/clash/component/auth"
	authStore "github.com/Dreamacro/clash/listener/auth"
	"net"

	"github.com/Dreamacro/clash/common/cache"
	N "github.com/Dreamacro/clash/common/net"
	C "github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/listener/http"
	"github.com/Dreamacro/clash/listener/socks"
	"github.com/Dreamacro/clash/transport/socks4"
	"github.com/Dreamacro/clash/transport/socks5"

	"crypto/tls"
	"github.com/quic-go/quic-go"
)

type Listener struct {
	listener net.Listener
	addr     string
	cache    *cache.LruCache
	closed   bool
}

// RawAddress implements C.Listener
func (l *Listener) RawAddress() string {
	return l.addr
}

// Address implements C.Listener
func (l *Listener) Address() string {
	return l.listener.Addr().String()
}

// Close implements C.Listener
func (l *Listener) Close() error {
	l.closed = true
	return l.listener.Close()
}

func New(addr string, in chan<- C.ConnContext) (*Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	ml := &Listener{
		listener: l,
		addr:     addr,
		cache:    cache.New(cache.WithAge(30)),
	}
	go func() {
		for {
			c, err := ml.listener.Accept()
			if err != nil {
				if ml.closed {
					break
				}
				continue
			}
			go handleConn(c, in, authStore.Authenticator())
		}
	}()

	return ml, nil
}

func NewQUIC(addr string, tlsConf *tls.Config, in chan<- C.ConnContext) (quic.Listener, error) {
	l, err := quic.ListenAddr(addr, tlsConf, nil)
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			var ctx = context.Background()
			c, err1 := l.Accept(ctx)
			if err1 != nil {
				continue
			}
			go func(conn quic.Connection) {
				for true {
					// Accept a stream
					stream, err := conn.AcceptStream(ctx)
					if err != nil {
						err = conn.Context().Err()
						if err != nil {
							break
						}
					}
					connStream := Conn{Connection: conn, Stream: stream}
					go handleQUIC(connStream, in, authStore.Authenticator())
				}
				err = conn.CloseWithError(0, "")
				if err != nil {
					return
				}
			}(c)
		}
	}()
	return l, nil
}

type Conn struct {
	quic.Connection
	quic.Stream
}

func (c Conn) Close() error {
	return c.Stream.Close()
}

func NewWithAuthenticate(addr string, in chan<- C.ConnContext, authenticator auth.Authenticator) (*Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	ml := &Listener{
		listener: l,
		addr:     addr,
		cache:    cache.New(cache.WithAge(30)),
	}
	go func() {
		for {
			c, err := ml.listener.Accept()
			if err != nil {
				if ml.closed {
					break
				}
				continue
			}
			go handleConn(c, in, authenticator)
		}
	}()

	return ml, nil
}

func handleConn(conn net.Conn, in chan<- C.ConnContext, authenticator auth.Authenticator) {
	conn.(*net.TCPConn).SetKeepAlive(true)

	bufConn := N.NewBufferedConn(conn)
	head, err := bufConn.Peek(1)
	if err != nil {
		return
	}

	switch head[0] {
	case socks4.Version:
		socks.HandleSocks4(bufConn, in, authenticator)
	case socks5.Version:
		socks.HandleSocks5(bufConn, in, authenticator)
	default:
		http.HandleConn(bufConn, in, authenticator)
	}
}

func handleQUIC(conn net.Conn, in chan<- C.ConnContext, authenticator auth.Authenticator) {
	bufConn := N.NewBufferedConn(conn)
	head, err := bufConn.Peek(1)
	if err != nil {
		return
	}
	switch head[0] {
	case socks4.Version:
		socks.HandleSocks4(bufConn, in, authenticator)
	case socks5.Version:
		socks.HandleSocks5(bufConn, in, authenticator)
	default:
		http.HandleConn(bufConn, in, authenticator)
	}
}

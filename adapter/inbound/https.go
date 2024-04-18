package inbound

import (
	C "github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/context"
	"net"
	"net/http"
)

// NewHTTPS receive CONNECT request and return ConnContext
func NewHTTPS(request *http.Request, conn net.Conn) *context.ConnContext {
	metadata := parseHTTPAddr(request)
	metadata.Type = C.HTTPCONNECT
	if ip, port, err := parseAddr(conn.RemoteAddr().String()); err == nil {
		metadata.SrcIP = ip
		metadata.SrcPort = port
	}
	if ip, port, err := parseAddr(conn.LocalAddr().String()); err == nil {
		metadata.LocIP = ip
		metadata.LocPort = port
	}
	//添加 context
	metadata.Ctx = request.Context()
	return context.NewConnContext(conn, metadata)
}

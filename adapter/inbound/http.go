package inbound

import (
	context2 "context"
	C "github.com/zhaofenghao/clash/constant"
	"github.com/zhaofenghao/clash/context"
	"github.com/zhaofenghao/clash/transport/socks5"
	"net"
)

// NewHTTP receive normal http request and return HTTPContext
func NewHTTP(target socks5.Addr, sourceConn net.Conn, conn net.Conn, ctx context2.Context) *context.ConnContext {
	metadata := parseSocksAddr(target)
	metadata.NetWork = C.TCP
	metadata.Type = C.HTTP
	if ip, port, err := parseAddr(sourceConn.RemoteAddr().String()); err == nil {
		metadata.SrcIP = ip
		metadata.SrcPort = port
	}
	if ip, port, err := parseAddr(sourceConn.LocalAddr().String()); err == nil {
		metadata.LocIP = ip
		metadata.LocPort = port
	}
	//添加 context
	metadata.Ctx = ctx
	return context.NewConnContext(conn, metadata)
}

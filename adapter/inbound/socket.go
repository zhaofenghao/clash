package inbound

import (
	context2 "context"
	C "github.com/zhaofenghao/clash/constant"
	"github.com/zhaofenghao/clash/context"
	"github.com/zhaofenghao/clash/transport/socks5"
	"net"
)

// NewSocket receive TCP inbound and return ConnContext
func NewSocket(target socks5.Addr, conn net.Conn, source C.Type, ctx context2.Context) *context.ConnContext {
	metadata := parseSocksAddr(target)
	metadata.NetWork = C.TCP
	metadata.Type = source
	if ip, port, err := parseAddr(conn.RemoteAddr().String()); err == nil {
		metadata.SrcIP = ip
		metadata.SrcPort = port
	}
	if ip, port, err := parseAddr(conn.LocalAddr().String()); err == nil {
		metadata.LocIP = ip
		metadata.LocPort = port
	}
	//添加 context
	metadata.Ctx = ctx
	return context.NewConnContext(conn, metadata)
}

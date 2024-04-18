package params

import (
	"context"
	"net"
)

type ValueContext struct {
	ctx      context.Context
	RemoteIp string
	LocalIp  string
}

func (v *ValueContext) WithContext(ctx context.Context) {
	if ctx == nil {
		panic("nil context")
	}
	v.ctx = ctx
}

func (v *ValueContext) SetRemoteIP(remoteAddr string) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		v.RemoteIp = net.ParseIP(host).String()
	}
}

func (v *ValueContext) WithValue(key, value any) {
	if v.ctx == nil {
		v.ctx = context.Background()
	}
	v.ctx = context.WithValue(v.ctx, key, value)
}

func (v *ValueContext) GetContext() context.Context {
	if v.ctx == nil {
		v.ctx = context.TODO()
	}
	return v.ctx
}

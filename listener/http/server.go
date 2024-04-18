package http

import (
	"github.com/zhaofenghao/clash/component/auth"
	authStore "github.com/zhaofenghao/clash/listener/auth"
	"net"

	C "github.com/zhaofenghao/clash/constant"
)

type Listener struct {
	listener net.Listener
	addr     string
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
	return NewWithAuthenticate(addr, in, authStore.Authenticator())

}

func NewWithAuthenticate(addr string, in chan<- C.ConnContext, authenticator auth.Authenticator) (*Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	hl := &Listener{
		listener: l,
		addr:     addr,
	}
	go func() {
		for {
			conn, err := hl.listener.Accept()
			if err != nil {
				if hl.closed {
					break
				}
				continue
			}
			go HandleConn(conn, in, authenticator)
		}
	}()

	return hl, nil
}

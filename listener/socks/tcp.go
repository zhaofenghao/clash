package socks

import (
	"context"
	"github.com/Dreamacro/clash/component/auth"
	authStore "github.com/Dreamacro/clash/listener/auth"
	"github.com/Dreamacro/clash/params"
	"io"
	"net"

	"github.com/Dreamacro/clash/adapter/inbound"
	N "github.com/Dreamacro/clash/common/net"
	C "github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/transport/socks4"
	"github.com/Dreamacro/clash/transport/socks5"

	"github.com/hashicorp/yamux"
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
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	sl := &Listener{
		listener: l,
		addr:     addr,
	}
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				if sl.closed {
					break
				}
				continue
			}
			go handleSocks(c, in, authStore.Authenticator())
		}
	}()

	return sl, nil
}

func NewYamux(addr string, in chan<- C.ConnContext) (*Listener, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	sl := &Listener{
		listener: l,
		addr:     addr,
	}
	go func() {
		for {
			c, err1 := l.Accept()
			if err1 != nil {
				if sl.closed {
					break
				}
				continue
			}
			c.(*net.TCPConn).SetKeepAlive(true)
			go func(conn net.Conn) {
				sessionServer, err := yamux.Server(conn, nil)
				if err != nil {
					return
				}
				for true {
					// Accept a stream
					stream, err := sessionServer.Accept()
					if err != nil {
						if sessionServer.IsClosed() {
							break
						}
						continue
					}
					go handleYamux(stream, in, authStore.Authenticator())
				}
				sessionServer.Close()
			}(c)
		}
	}()
	return sl, nil
}
func handleYamux(conn net.Conn, in chan<- C.ConnContext, authenticator auth.Authenticator) {
	bufConn := N.NewBufferedConn(conn)
	head, err := bufConn.Peek(1)
	if err != nil {
		conn.Close()
		return
	}

	switch head[0] {
	case socks4.Version:
		HandleSocks4(bufConn, in, authenticator)
	case socks5.Version:
		HandleSocks5(bufConn, in, authenticator)
	default:
		conn.Close()
	}
}

func handleSocks(conn net.Conn, in chan<- C.ConnContext, authenticator auth.Authenticator) {
	conn.(*net.TCPConn).SetKeepAlive(true)
	bufConn := N.NewBufferedConn(conn)
	head, err := bufConn.Peek(1)
	if err != nil {
		conn.Close()
		return
	}

	switch head[0] {
	case socks4.Version:
		HandleSocks4(bufConn, in, authenticator)
	case socks5.Version:
		HandleSocks5(bufConn, in, authenticator)
	default:
		conn.Close()
	}
}

func HandleSocks4(conn net.Conn, in chan<- C.ConnContext, authenticator auth.Authenticator) {
	addr, _, err := socks4.ServerHandshake(conn, authenticator)
	if err != nil {
		conn.Close()
		return
	}
	in <- inbound.NewSocket(socks5.ParseAddr(addr), conn, C.SOCKS4, context.TODO())
}

func HandleSocks5(conn net.Conn, in chan<- C.ConnContext, authenticator auth.Authenticator) {
	var ctx = new(params.ValueContext)
	ctx.SetRemoteIP(conn.RemoteAddr().String())
	target, command, err := socks5.ServerHandshake(ctx, conn, authenticator)
	if err != nil {
		conn.Close()
		return
	}
	if command == socks5.CmdUDPAssociate {
		defer conn.Close()
		io.Copy(io.Discard, conn)
		return
	}
	in <- inbound.NewSocket(target, conn, C.SOCKS5, ctx.GetContext())
}

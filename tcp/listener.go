package tcp

import (
	"github.com/hwcer/cosnet/listener"
	"net"
)

func New(network, address string) (listener.Listener, error) {
	ln, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	return &Listener{Listener: ln}, nil
}

type Listener struct {
	net.Listener
}

func (ln *Listener) Accept() (listener.Conn, error) {
	conn, err := ln.Listener.Accept()
	if err == nil {
		return NewConn(conn), nil
	}
	return nil, err
}

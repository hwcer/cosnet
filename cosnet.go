package cosnet

import (
	"errors"
	"fmt"
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/cosnet/tcp"
	"github.com/hwcer/logger"
	"net"
	"strings"
	"time"
)

// Start 启动服务器,监听address
func (this *Server) Start(address string) (listener listener.Listener, err error) {
	addr := utils.NewAddress(address)
	if addr.Scheme == "" {
		addr.Scheme = "tcp"
	}
	network := strings.ToLower(addr.Scheme)
	switch network {
	case "tcp", "tcp4", "tcp6":
		listener, err = tcp.New(network, addr.String())
	//case "udp", "udp4", "udp6":
	//	listener, err = udp.New(network, addr.String())
	//case "unix", "unixgram", "unixpacket":
	default:
		err = errors.New("address scheme unknown")
	}
	if err == nil {
		this.Accept(listener)
	}
	return
}
func (this *Server) Accept(ln listener.Listener) {
	this.listener = append(this.listener, ln)
	go func() {
		this.accept(ln)
	}()
}
func (this *Server) accept(ln listener.Listener) {
	defer func() {
		if err := recover(); err != nil {
			logger.Alert(err)
		}
	}()
	defer func() {
		_ = ln.Close()
	}()
	for !this.SCC.Stopped() {
		conn, err := ln.Accept()
		if err == nil {
			_, err = this.New(conn)
		}
		if err != nil && !errors.Is(err, net.ErrClosed) && !this.SCC.Stopped() {
			logger.Error("tcp listener.Accept Error:%v", err)
		}
	}
}

// Connect 连接服务器address
func (this *Server) Connect(address string) (socket *Socket, err error) {
	conn, err := this.tryConnect(address)
	if err != nil {
		return nil, err
	}
	return this.New(conn)
}

func (this *Server) tryConnect(s string) (listener.Conn, error) {
	address := utils.NewAddress(s)
	if address.Scheme == "" {
		address.Scheme = "tcp"
	}
	rs := address.String()
	for try := uint16(0); try < Options.ClientReconnectMax; try++ {
		conn, err := net.DialTimeout(address.Scheme, rs, time.Second)
		if err == nil {
			return tcp.NewConn(conn), nil
		} else {
			fmt.Printf("%v %v\n", try, err)
			time.Sleep(time.Duration(Options.ClientReconnectTime))
		}
	}
	return nil, fmt.Errorf("failed to dial %v", rs)
}

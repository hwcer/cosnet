package cosnet

import (
	"errors"
	"fmt"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/cosnet/tcp"
	"github.com/hwcer/cosnet/udp"
	"github.com/hwcer/logger"
	"net"
	"strings"
	"time"
)

// Listen 启动柜服务器,监听address
func (this *Server) Listen(address string) (listener net.Listener, err error) {
	addr := utils.NewAddress(address)
	if addr.Scheme == "" {
		return nil, errors.New("address scheme empty")
	}
	network := strings.ToLower(addr.Scheme)
	switch network {
	case "tcp", "tcp4", "tcp6":
		listener, err = tcp.New(network, addr.String())
	case "udp", "udp4", "udp6":
		listener, err = udp.New(network, addr.String())
	//case "unix", "unixgram", "unixpacket":
	default:
		err = errors.New("address scheme unknown")
	}
	if err == nil {
		this.Accept(listener)
		this.listener = append(this.listener, listener)
	}
	return
}
func (this *Server) Accept(ln net.Listener) {
	scc.GO(func() {
		this.accept(ln)
	})
}
func (this *Server) accept(ln net.Listener) {
	defer func() {
		if err := recover(); err != nil {
			logger.Alert(err)
		}
	}()
	defer func() {
		_ = ln.Close()
	}()
	for !scc.Stopped() {
		conn, err := ln.Accept()
		if err == nil {
			_, err = this.New(conn)
		}
		if err != nil && !errors.Is(err, net.ErrClosed) && !scc.Stopped() {
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

func (this *Server) tryConnect(s string) (net.Conn, error) {
	address := utils.NewAddress(s)
	if address.Scheme == "" {
		address.Scheme = "tcp"
	}
	rs := address.String()
	for try := uint16(0); try < Options.ClientReconnectMax; try++ {
		conn, err := net.DialTimeout(address.Scheme, rs, time.Second)
		if err == nil {
			return conn, nil
		} else {
			fmt.Printf("%v %v\n", try, err)
			time.Sleep(time.Duration(Options.ClientReconnectTime))
		}
	}
	return nil, fmt.Errorf("failed to dial %v", rs)
}

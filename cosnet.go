package cosnet

import (
	"context"
	"errors"
	"fmt"
	"github.com/hwcer/cosgo/logger"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/cosnet/message"
	"github.com/hwcer/cosnet/tcp"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"
)

var started int32

//func Matcher() cmux.Matcher {
//	magic := message.Options.MagicNumber
//	return
//}

func Matcher(r io.Reader) bool {
	buf := make([]byte, 1)
	n, _ := r.Read(buf)
	return n == 1 && (buf[0] == message.MagicNumber || buf[0] == message.MagicConfirm)
}

// Listen 监听address
func Listen(address string) (listener listener.Listener, err error) {
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
		Accept(listener)
		instance = append(instance, listener)
	}

	return
}
func Accept(ln listener.Listener) {
	scc.CGO(func(ctx context.Context) {
		acceptListener(ln)
	})
	Start()
}

func Start() bool {
	if atomic.CompareAndSwapInt32(&started, 0, 1) {
		scc.CGO(heartbeat)
		scc.Trigger(stop)
		return true
	}
	return false
}

func stop() {
	for _, l := range instance {
		_ = l.Close()
	}
}

func acceptListener(ln listener.Listener) {
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
			_, err = New(conn)
		}
		if err != nil && !errors.Is(err, net.ErrClosed) && !scc.Stopped() {
			logger.Error("tcp listener.Accept Error:%v", err)
		}
	}
}

// Connect 连接服务器address
func Connect(address string) (socket *Socket, err error) {
	conn, err := tryConnect(address)
	if err != nil {
		return nil, err
	}
	Start()
	return New(conn)
}

func tryConnect(s string) (listener.Conn, error) {
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

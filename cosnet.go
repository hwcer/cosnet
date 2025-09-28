package cosnet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/hwcer/cosgo"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/cosnet/message"
	"github.com/hwcer/cosnet/tcp"
	"github.com/hwcer/logger"
)

var lns []listener.Listener

func init() {
	cosgo.On(cosgo.EventTypStarted, onStart)
	cosgo.On(cosgo.EventTypClosing, onClose)
}

func Matcher(r io.Reader) bool {
	buf := make([]byte, 1)
	n, _ := r.Read(buf)
	return n == 1 && message.Magics.Has(buf[0])
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
}

func onStart() error {
	scc.CGO(heartbeat)
	return nil
}

func onClose() error {
	for _, ln := range lns {
		_ = ln.Close()
	}
	return nil
}

// 启动服务器应该在初始化时的主进程中完成
// 不应该是在并发的协程中完成
func acceptListener(ln listener.Listener) {
	defer func() {
		if err := recover(); err != nil {
			logger.Alert(err)
		}
	}()
	defer func() {
		_ = ln.Close()
	}()
	lns = append(lns, ln)
	for !scc.Stopped() {
		conn, err := ln.Accept()
		if err == nil {
			_, err = New(conn)
		} else if errors.Is(err, net.ErrClosed) {
			return
		} else {
			logger.Alert("listener.Accept Error:%v", err)
		}
	}
}

// Connect 连接服务器address
func Connect(address string) (socket *Socket, err error) {
	conn, err := tryConnect(address)
	if err != nil {
		return nil, err
	}
	return New(conn)
}

func tryConnect(s string) (listener.Conn, error) {
	address := utils.NewAddress(s)
	if address.Scheme == "" {
		address.Scheme = "tcp"
	}
	rs := address.String()
	for try := int32(0); try < Options.ClientReconnectMax; try++ {
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

package cosnet

import (
	"github.com/hwcer/cosgo/logger"
	"github.com/hwcer/cosnet/sockets"
	"net"
)

func NewTcpServer(engine *sockets.Engine, address string) (listener net.Listener, err error) {
	listener, err = net.Listen("tcp", address)
	if err != nil {
		return
	}
	engine.SCC().GO(func() {
		startTcpServer(engine, listener)
	})
	return
}

func startTcpServer(engine *sockets.Engine, listener net.Listener) {
	defer func() {
		_ = listener.Close()
		if err := recover(); err != nil {
			logger.Error(err)
		}
	}()
	for !engine.SCC().Stopped() {
		if conn, err := listener.Accept(); err == nil {
			engine.New(conn, sockets.NetTypeServer)
		} else {
			logger.Error("listener.Accept Error:%v", err)
		}
	}
}

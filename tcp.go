package cosnet

import (
	"github.com/hwcer/cosnet/sockets"
	"github.com/hwcer/logger"
	"net"
)

func NewTcpServer(agents *sockets.Agents, address string) (listener net.Listener, err error) {
	listener, err = net.Listen("tcp", address)
	if err != nil {
		return
	}
	agents.SCC().GO(func() {
		startTcpServer(agents, listener)
	})
	return
}

func startTcpServer(agents *sockets.Agents, listener net.Listener) {
	defer func() {
		_ = listener.Close()
		if err := recover(); err != nil {
			logger.Error(err)
		}
	}()
	for !agents.SCC().Stopped() {
		conn, err := listener.Accept()
		if err == nil {
			_, err = agents.New(conn, sockets.NetTypeServer)
		}
		if err != nil {
			logger.Error("listener.Accept Error:%v", err)
		}
	}
}

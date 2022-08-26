package cosnet

import (
	"github.com/hwcer/cosnet/sockets"
	"github.com/hwcer/cosnet/udp"
	"github.com/hwcer/logger"
	"net"
)

func (this *Cosnet) NewTcpServer(network, address string) (listener net.Listener, err error) {
	listener, err = net.Listen(network, address)
	if err != nil {
		return
	}
	this.Agents.GO(func() {
		this.tcpListener(listener)
	})
	return
}

func (this *Cosnet) NewUdpServer(network, address string) (server *udp.Server, err error) {
	server, err = udp.New(this.Agents, network, address)
	return
}

func (this *Cosnet) tcpListener(listener net.Listener) {
	defer func() {
		_ = listener.Close()
		if err := recover(); err != nil {
			logger.Error(err)
		}
	}()
	for !this.Agents.Stopped() {
		conn, err := listener.Accept()
		if err == nil {
			_, err = this.Agents.New(conn, sockets.NetTypeServer)
		}
		if err != nil {
			logger.Error("listener.Accept Error:%v", err)
		}
	}
}

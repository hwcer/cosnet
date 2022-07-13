package cosnet

import (
	"github.com/hwcer/cosgo/logger"
	"net"
)

type NetType uint8

//NetType
const (
	NetTypeClient NetType = 1 //client Request
	NetTypeServer         = 2 //Server Listener
)

type Server struct {
	cosnet   *Cosnet
	address  string
	listener net.Listener
}

func NewServer(cosnet *Cosnet, address string) (*Server, error) {
	server := &Server{cosnet: cosnet, address: address}
	if listener, err := net.Listen("tcp", address); err != nil {
		return nil, err
	} else {
		server.listener = listener
	}
	return server, nil
}

//Start listener.Accept
func (this *Server) Start() {
	for !this.cosnet.scc.Stopped() {
		if conn, err := this.listener.Accept(); err == nil {
			this.cosnet.New(conn, NetTypeServer)
		} else {
			logger.Error("listener.Accept Error:%v", err)
		}
	}
}

func (this *Server) Close() error {
	return this.listener.Close()
}

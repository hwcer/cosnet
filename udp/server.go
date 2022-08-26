package udp

import (
	"bytes"
	"context"
	"github.com/hwcer/cosnet/sockets"
	"net"
	"sync"
)

type Server struct {
	dict        map[string]*Conn
	addr        *net.UDPAddr
	conn        *net.UDPConn
	agents      *sockets.Agents
	dictMutex   sync.Mutex
	listenMutex sync.Mutex
}

func New(agents *sockets.Agents, network, address string) (server *Server, err error) {
	server = &Server{dict: make(map[string]*Conn), agents: agents}
	server.addr, err = net.ResolveUDPAddr(network, address)
	if err != nil {
		return
	}
	server.conn, err = net.ListenUDP(network, server.addr)
	if err != nil {
		return
	}
	if err = server.conn.SetReadBuffer(1 << 24); err != nil {
		return
	}
	if err = server.conn.SetWriteBuffer(1 << 24); err != nil {
		return
	}
	err = server.start()
	return
}

func (this *Server) start() error {
	for i := 0; i < sockets.Options.UdpServerWorker; i++ {
		this.agents.GO(func() {
			this.listen()
		})
	}
	this.agents.CGO(this.close)
	return nil
}

func (this *Server) close(ctx context.Context) {
	defer func() {
		_ = this.conn.Close()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}

func (this *Server) LoadOrStore(addr *net.UDPAddr) (conn *Conn) {
	this.dictMutex.Lock()
	defer this.dictMutex.Unlock()
	name := addr.String()
	if conn = this.dict[name]; conn != nil {
		return
	}
	conn = &Conn{}
	conn.addr = addr
	conn.server = this
	conn.buffer = &bytes.Buffer{}
	if _, err := this.agents.New(conn, sockets.NetTypeServer); err != nil {
		this.dict[name] = conn
	}
	return
}

func (this *Server) Delete(addr *net.UDPAddr) {
	this.dictMutex.Lock()
	defer this.dictMutex.Unlock()
	delete(this.dict, addr.String())
}

func (this *Server) listen() {
	data := make([]byte, 1<<16)
	for !this.agents.Stopped() {
		this.listenMutex.Lock()
		n, addr, err := this.conn.ReadFromUDP(data)
		this.listenMutex.Unlock()
		if err != nil {
			if err.(net.Error).Timeout() {
				continue
			}
			break
		}
		if n <= 0 {
			continue
		}
		conn := this.LoadOrStore(addr)
		conn.Lock()
		_, _ = conn.buffer.Write(data[0:n])
		conn.Unlock()
	}
}

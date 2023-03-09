package cosnet

import (
	"bytes"
	"context"
	"net"
	"sync"
	"time"
)

// udpConn net.Conn
type udpConn struct {
	sync.Mutex
	addr   *net.UDPAddr
	server *udpServer
	buffer *bytes.Buffer
}

func (this *udpConn) Read(b []byte) (n int, err error) {
	return this.buffer.Read(b)
}
func (this *udpConn) Write(b []byte) (n int, err error) {
	return this.server.conn.WriteToUDP(b, this.addr)
}

func (this *udpConn) Close() error {
	this.server.delete(this.addr)
	return nil
}

func (this *udpConn) LocalAddr() net.Addr {
	return this.server.conn.LocalAddr()
}

func (this *udpConn) RemoteAddr() net.Addr {
	return this.addr
}

func (this *udpConn) SetDeadline(t time.Time) error {
	return nil
}

func (this *udpConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (this *udpConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type udpServer struct {
	dict        map[string]*udpConn
	addr        *net.UDPAddr
	conn        *net.UDPConn
	agents      *Server
	dictMutex   sync.Mutex
	listenMutex sync.Mutex
}

func (this *udpServer) start() error {
	for i := 0; i < Options.UdpServerWorker; i++ {
		this.agents.GO(func() {
			this.listen()
		})
	}
	this.agents.CGO(this.close)
	return nil
}

func (this *udpServer) close(ctx context.Context) {
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

func (this *udpServer) LoadOrStore(addr *net.UDPAddr) (conn *udpConn) {
	this.dictMutex.Lock()
	defer this.dictMutex.Unlock()
	name := addr.String()
	if conn = this.dict[name]; conn != nil {
		return
	}
	conn = &udpConn{}
	conn.addr = addr
	conn.server = this
	conn.buffer = &bytes.Buffer{}
	if _, err := this.agents.New(conn, NetTypeServer); err != nil {
		this.dict[name] = conn
	}
	return
}

func (this *udpServer) delete(addr *net.UDPAddr) {
	this.dictMutex.Lock()
	defer this.dictMutex.Unlock()
	delete(this.dict, addr.String())
}

func (this *udpServer) listen() {
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

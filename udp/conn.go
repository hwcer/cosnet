package udp

import (
	"bytes"
	"net"
	"sync"
	"time"
)

// net.Conn
type Conn struct {
	sync.Mutex
	addr   *net.UDPAddr
	server *Server
	buffer *bytes.Buffer
}

func (this *Conn) Read(b []byte) (n int, err error) {
	return this.buffer.Read(b)
}
func (this *Conn) Write(b []byte) (n int, err error) {
	return this.server.conn.WriteToUDP(b, this.addr)
}

func (this *Conn) Close() error {
	this.server.Delete(this.addr)
	return nil
}

func (this *Conn) LocalAddr() net.Addr {
	return this.server.conn.LocalAddr()
}

func (this *Conn) RemoteAddr() net.Addr {
	return this.addr
}

func (this *Conn) SetDeadline(t time.Time) error {
	return nil
}

func (this *Conn) SetReadDeadline(t time.Time) error {
	return nil
}

func (this *Conn) SetWriteDeadline(t time.Time) error {
	return nil
}

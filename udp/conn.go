package udp

import (
	"bytes"
	"net"
	"sync"
	"time"
)

// Conn net.Conn
type Conn struct {
	addr     *net.UDPAddr
	buffer   *bytes.Buffer
	listener *Listener
	sync.Mutex
}

func (this *Conn) Read(b []byte) (n int, err error) {
	return this.buffer.Read(b)
}
func (this *Conn) Write(b []byte) (n int, err error) {
	return this.listener.conn.WriteToUDP(b, this.addr)
}

func (this *Conn) Close() error {
	this.listener.delete(this.addr)
	return nil
}

func (this *Conn) LocalAddr() net.Addr {
	return this.listener.conn.LocalAddr()
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

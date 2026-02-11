package listener

import (
	"net"

	"github.com/hwcer/cosgo/session"
	"github.com/hwcer/cosnet/message"
)

type SocketType int8

const (
	SocketTypeNone SocketType = iota
	SocketTypeClient
	SocketTypeServer
)

type Conn interface {
	net.Conn
	ReadMessage(Socket, message.Message) error
	WriteMessage(Socket, message.Message) error
}

type Listener interface {
	// Accept waits for and returns the next connection to the listener.
	Accept() (Conn, error)

	// Close closes the listener.
	// Any blocked Accept operations will be unblocked and return errors.
	Close() error

	// Addr returns the listener's network address.
	Addr() net.Addr
}

type Socket interface {
	Id() uint64
	Type() SocketType
	Data() *session.Data
	Conn() Conn
	Send(flag message.Flag, index int32, path string, data any)
	Errorf(format any, args ...any)
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

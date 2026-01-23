package listener

import (
	"net"

	"github.com/hwcer/cosnet/message"
)

type Conn interface {
	net.Conn
	ReadMessage(message.Message) error
	WriteMessage(message.Message) error
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

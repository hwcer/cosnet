package tcp

import (
	"net"
)

func New(network, address string) (ln net.Listener, err error) {
	ln, err = net.Listen(network, address)
	return
}

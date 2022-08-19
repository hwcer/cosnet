package cosnet

import (
	"errors"
	"fmt"
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/cosnet/sockets"
	"net"
	"time"
)

// cliConnect create client create
func cliConnect(agents *sockets.Agents, address string) (*sockets.Socket, error) {
	conn, err := tryConnect(address)
	if err != nil {
		return nil, err
	}
	return agents.New(conn, sockets.NetTypeClient)
}

func tryConnect(s string) (net.Conn, error) {
	address := utils.NewAddress(s)
	if address.Scheme == "" {
		address.Scheme = "tcp"
	}
	rs := address.String()
	for try := uint16(0); try < sockets.Options.ClientReconnectMax; try++ {
		conn, err := net.DialTimeout(address.Scheme, rs, time.Second)
		if err == nil {
			return conn, nil
		} else {
			fmt.Printf("%v %v\n", try, err)
			time.Sleep(time.Duration(sockets.Options.ClientReconnectTime))
		}
	}
	return nil, errors.New("Failed to create to Server")
}

package cosnet

import (
	"errors"
	"fmt"
	"net"
	"time"
)

//NewConnect create client create
func NewConnect(cosnet *Cosnet, address string) (*Socket, error) {
	conn, err := tryConnect(address)
	if err != nil {
		return nil, err
	}
	return cosnet.New(conn, NetTypeClient)
}

func tryConnect(address string) (net.Conn, error) {
	for try := uint16(0); try <= Options.ClientReconnectMax; try++ {
		conn, err := net.DialTimeout("tcp", address, time.Second)
		if err == nil {
			return conn, nil
		} else {
			fmt.Printf("%v create error:%v\n", try, err)
			time.Sleep(time.Duration(Options.ClientReconnectTime))
		}
	}
	return nil, errors.New("Failed to create to Server")
}

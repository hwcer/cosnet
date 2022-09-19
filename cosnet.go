package cosnet

import (
	"errors"
	"fmt"
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/logger"
	"net"
	"strings"
	"time"
)

// Listen 启动柜服务器,监听address
func (this *Agents) Listen(address string) (listener interface{}, err error) {
	addr := utils.NewAddress(address)
	if addr.Scheme == "" {
		return nil, errors.New("address scheme empty")
	}
	network := strings.ToLower(addr.Scheme)
	switch network {
	case "tcp", "tcp4", "tcp6":
		listener, err = this.NewTcpServer(network, addr.String())
	case "udp", "udp4", "udp6":
		listener, err = this.NewUdpServer(network, addr.String())
	//case "unix", "unixgram", "unixpacket":
	default:
		err = errors.New("address scheme unknown")
	}
	return
}

// Connect 连接服务器address
func (this *Agents) Connect(address string) (socket *Socket, err error) {
	conn, err := this.tryConnect(address)
	if err != nil {
		return nil, err
	}
	return this.New(conn, NetTypeClient)
}

func (this *Agents) NewTcpServer(network, address string) (listener net.Listener, err error) {
	listener, err = net.Listen(network, address)
	if err != nil {
		return
	}
	this.GO(func() {
		this.tcpListener(listener)
	})
	return
}

func (this *Agents) NewUdpServer(network, address string) (server *udpServer, err error) {
	server = &udpServer{dict: make(map[string]*udpConn), agents: this}
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

func (this *Agents) tcpListener(listener net.Listener) {
	defer func() {
		_ = listener.Close()
		if err := recover(); err != nil {
			logger.Error(err)
		}
	}()
	for !this.Stopped() {
		conn, err := listener.Accept()
		if err == nil {
			_, err = this.New(conn, NetTypeServer)
		}
		if err != nil {
			logger.Error("listener.Accept Error:%v", err)
		}
	}
}

func (this *Agents) tryConnect(s string) (net.Conn, error) {
	address := utils.NewAddress(s)
	if address.Scheme == "" {
		address.Scheme = "tcp"
	}
	rs := address.String()
	for try := uint16(0); try < Options.ClientReconnectMax; try++ {
		conn, err := net.DialTimeout(address.Scheme, rs, time.Second)
		if err == nil {
			return conn, nil
		} else {
			fmt.Printf("%v %v\n", try, err)
			time.Sleep(time.Duration(Options.ClientReconnectTime))
		}
	}
	return nil, errors.New("Failed to create to udpServer")
}

package cosnet

import (
	"bytes"
	"github.com/hwcer/cosnet/sockets"
	"net"
	"sync"
	"sync/atomic"
)

var udpMap = sync.Map{}
var udpLock sync.Mutex

func (this *Cosnet) NewUdpServer(network, address string) (listener *net.UDPConn, err error) {
	var udpAddress *net.UDPAddr
	udpAddress, err = net.ResolveUDPAddr(network, address)
	if err != nil {
		return nil, err
	}
	listener, err = net.ListenUDP(network, udpAddress)
	if err != nil {
		return
	}
	for i := 0; i < sockets.Options.UdpServerWorker; i++ {
		this.Agents.GO(func() {
			this.udpListen(listener)
		})
	}
	this.Agents.GO(func() {
		this.udpCloser(listener)
	})
	return
}

type udpSocket struct {
	sync.Mutex
	addr   *net.UDPAddr
	conn   *net.UDPConn //连接
	buffer *bytes.Buffer
}

func (this *Cosnet) udpListen(conn *net.UDPConn) {
	data := make([]byte, 1<<16)
	for !this.Agents.Stopped() {
		udpLock.Lock()
		n, addr, err := conn.ReadFromUDP(data)
		udpLock.Unlock()
		if err != nil {
			if err.(net.Error).Timeout() {
				continue
			}
			break
		}
		if n <= 0 {
			continue
		}
		addrStr := addr.String()
		socket := &udpSocket{}
		socket.Mutex.Lock()
		if v, loaded := udpMap.LoadOrStore(addrStr, socket); loaded {
			socket, _ = v.(*udpSocket)
		} else {
			socket.addr = addr
			socket.conn = conn
			socket.buffer = bytes.Buffer{}
		}

		udpLock.Lock()
		helper, ok := udpMap[addrStr]
		if !ok {
			helper = &udpMsgQueHelper{null: true}
			udpMap[addrStr] = helper
		}
		udpLock.Unlock()

		if helper.null {
			helper.Lock()
			if atomic.CompareAndSwapInt32(&helper.init, 0, 1) {
				helper.udpMsgQue = newUdpAccept(r.conn, r.msgTyp, r.handler, r.parserFactory, addr)
				helper.null = false
			}
			helper.Unlock()
		}

		if !helper.sendRead(data, n) {
			LogError("drop msg because msgque full msgqueid:%v", helper.id)
		}
	}
}

func (this *Cosnet) udpCloser(conn *net.UDPConn) {
	defer func() {
		_ = conn.Close()
	}()
	scc := this.Agents.SCC()
	for {
		select {
		case <-scc.Context.Done():
			return
		}
	}
}

package udp

import (
	"bytes"
	"context"
	"github.com/hwcer/cosgo/scc"
	"net"
	"sync"
)

func New(network, address string) (net.Listener, error) {
	var err error
	ln := &Listener{}
	ln.sockets = sync.Map{}
	ln.accepts = make(chan *Conn, Options.acceptsCap)
	ln.addr, err = net.ResolveUDPAddr(network, address)
	if err != nil {
		return nil, err
	}
	ln.conn, err = net.ListenUDP(network, ln.addr)
	if err != nil {
		return nil, err
	}
	if err = ln.conn.SetReadBuffer(1 << 24); err != nil {
		return nil, err
	}
	if err = ln.conn.SetWriteBuffer(1 << 24); err != nil {
		return nil, err
	}
	err = ln.Start()
	return ln, nil
}

type Listener struct {
	addr        *net.UDPAddr
	conn        *net.UDPConn
	stop        chan struct{}
	sockets     sync.Map
	accepts     chan *Conn
	listenMutex sync.Mutex
}

func (this *Listener) Addr() net.Addr {
	return this.addr
}

func (this *Listener) Close() (err error) {
	defer func() {
		_ = recover()
	}()
	close(this.stop)
	return nil
}

func (this *Listener) Accept() (ln net.Conn, err error) {
	select {
	case ln = <-this.accepts:
	case <-this.stop:
		err = net.ErrClosed
	case <-scc.Context().Done():
		err = net.ErrClosed
	}
	return
}

func (this *Listener) Start() error {
	if this.stop != nil {
		return nil
	}
	this.stop = make(chan struct{})
	for i := 0; i < Options.ServerWorker; i++ {
		scc.GO(func() {
			this.listen()
		})
	}
	scc.CGO(this.close)
	return nil
}

func (this *Listener) close(ctx context.Context) {
	defer func() {
		_ = this.conn.Close()
	}()
	for {
		select {
		case <-this.stop:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (this *Listener) LoadOrStore(addr *net.UDPAddr) (conn *Conn) {
	key := addr.String()
	conn = &Conn{}
	conn.Lock()
	if i, loaded := this.sockets.LoadOrStore(key, conn); loaded {
		conn = i.(*Conn)
		conn.Lock()
	} else {
		conn.addr = addr
		conn.buffer = &bytes.Buffer{}
		conn.listener = this
		select {
		case this.accepts <- conn:
		default:
			this.sockets.Delete(key)
		}
	}
	return
}

func (this *Listener) delete(addr *net.UDPAddr) {
	this.sockets.Delete(addr.String())
}

func (this *Listener) listen() {
	data := make([]byte, 1<<16)
	for !scc.Stopped() {
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
		_, _ = conn.buffer.Write(data[0:n])
		conn.Unlock()
	}
}

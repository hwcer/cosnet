package cosnet

import (
	"context"
	"github.com/hwcer/cosgo/storage"
	"github.com/hwcer/cosgo/values"
	"io"
	"net"
)

func NewSocket(srv *Server, conn net.Conn) *Socket {
	socket := &Socket{conn: conn, server: srv}
	socket.stop = make(chan struct{})
	//socket.status = NewStatus()
	socket.cwrite = make(chan *Message, Options.WriteChanSize)
	socket.server.scc.SGO(socket.readMsg, socket.Errorf)
	socket.server.scc.SGO(socket.writeMsg, socket.Errorf)
	return socket
}

// Socket 基础网络连接
type Socket struct {
	storage.Data
	conn   net.Conn
	stop   chan struct{}
	server *Server
	status Status
	cwrite chan *Message //写入通道,仅仅强制关闭的会被CLOSE
	//netType NetType       //网络连接类型
}

func (this *Socket) emit(e EventType) bool {
	return this.server.Emit(e, this)
}

func (this *Socket) close() {
	defer func() { _ = recover() }()
	if this.stop != nil {
		close(this.stop)
	}
	_ = this.conn.Close()
}

// disconnect 掉线,包含网络超时，网络错误
func (this *Socket) disconnect() {
	if this.status.Disconnect() {
		this.close()
		this.KeepAlive()
		this.emit(EventTypeDisconnect)
	}
}

// Close 强制关闭,无法重连
func (this *Socket) Close(msg ...*Message) {
	if !this.status.Close() || len(msg) == 0 {
		return
	}
	defer func() {
		_ = recover()
	}()
	for _, m := range msg {
		_ = this.write(m)
	}
}

func (this *Socket) Player() (r *Player) {
	v := this.Data.Get()
	if v != nil {
		r, _ = v.(*Player)
	}
	return
}

func (this *Socket) Errorf(format any) {
	this.server.Errorf(this, format)
}

// Verified 是否已经登录
func (this *Socket) Verified() bool {
	return this.Get() != nil
}

// Heartbeat 每一次Heartbeat() heartbeat计数加1
func (this *Socket) Heartbeat() {
	heartbeat := this.status.Heartbeat()
	switch this.status.status {
	case StatusTypeDestroyed:
		this.server.Remove(this)
	case StatusTypeDisconnect:
		if !this.Verified() || Options.SocketReconnectTime == 0 || heartbeat > Options.SocketReconnectTime {
			this.server.Remove(this)
		}
	default:
		if this.status.closing && (len(this.cwrite) == 0 || heartbeat >= Options.SocketDestroyingTime) {
			this.close()
		}
		if heartbeat >= Options.SocketConnectTime {
			this.disconnect()
		}
	}
}

// KeepAlive 任何行为都清空heartbeat
func (this *Socket) KeepAlive() {
	this.status.KeepAlive()
}

func (this *Socket) LocalAddr() net.Addr {
	if this.conn != nil {
		return this.conn.LocalAddr()
	}
	return nil
}
func (this *Socket) RemoteAddr() net.Addr {
	if this.conn != nil {
		return this.conn.RemoteAddr()
	}
	return nil
}

func (this *Socket) Send(code int16, path string, data any) (err error) {
	m := Pool.Acquire()
	defer func() {
		if err != nil {
			Pool.Release(m)
		}
	}()
	if err = m.Marshal(code, path, data, nil); err != nil {
		return
	}
	err = this.Write(m)
	return
}

// Write 外部写入消息,慎用,注意发送失败时消息回收,参考Send
func (this *Socket) Write(m *Message) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = values.Error(e)
		}
	}()
	if this.status.Disabled() {
		return ErrSocketClosed
	}
	if !this.write(m) {
		err = ErrSocketChannelFull
	}
	return
}

func (this *Socket) write(m *Message) bool {
	if m == nil || this.cwrite == nil {
		return true
	}
	select {
	case this.cwrite <- m:
		return true
	default:
		return false
	}
}

func (this *Socket) processMsg(socket *Socket, msg *Message) {
	this.KeepAlive()
	this.server.handle(socket, msg)
}

func (this *Socket) readMsg(ctx context.Context) {
	defer this.disconnect()
	var err error
	head := make([]byte, MessageHead)
	for {
		_, err = io.ReadFull(this.conn, head)
		if err != nil {
			this.Errorf(err)
			return
		}
		if !this.readMsgTrue(head) {
			return
		}
	}
}

func (this *Socket) writeMsg(ctx context.Context) {
	defer this.disconnect()
	var msg *Message
	for {
		select {
		case <-ctx.Done():
			return
		case <-this.stop:
			return
		case msg = <-this.cwrite:
			if !this.writeMsgTrue(msg) {
				return
			}
		}
	}
}

func (this *Socket) readMsgTrue(head []byte) (r bool) {
	//logger.Debug("READ HEAD:%v", head)
	msg := Pool.Acquire()
	defer func() {
		Pool.Release(msg)
	}()
	err := msg.Parse(head)
	if err != nil {
		this.server.Errorf(this, "READ HEAD ERR,RemoteAddr:%v,HEAD:%v", err, this.RemoteAddr().String(), head)
		return false
	}
	//logger.Debug("READ HEAD:%+v BYTE:%v", *msg.Header, head)
	if msg.Len() > 0 {
		_, err = msg.Write(this.conn)
		if err != nil {
			this.server.Errorf(this, "READ BODY ERR:%v", err)
			return false
		}
	}
	this.processMsg(this, msg)
	return true
}

func (this *Socket) writeMsgTrue(msg *Message) (r bool) {
	defer func() {
		Pool.Release(msg)
	}()
	data, err := msg.Bytes()
	if err != nil {
		this.Errorf(err)
		return true
	}
	//logger.Debug("socket write error,code:%v, data:%v", msg.Code(), msg.data())
	var n int
	writeCount := 0
	for writeCount < len(data) {
		n, err = this.conn.Write(data[writeCount:])
		if err != nil {
			this.Errorf(err)
			return false
		}
		writeCount += n
	}
	this.KeepAlive()
	return true
}

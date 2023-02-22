package cosnet

import (
	"context"
	"github.com/hwcer/cosgo/storage"
	"github.com/hwcer/cosgo/values"
	"io"
	"net"
)

func NewSocket(engine *Sockets, conn net.Conn, netType NetType) *Socket {
	socket := &Socket{conn: conn, Sockets: engine, netType: netType}
	socket.stop = make(chan struct{})
	socket.Status = NewStatus()
	socket.cwrite = make(chan *Message, Options.WriteChanSize)
	socket.Sockets.scc.SGO(socket.readMsg, socket.Errorf)
	socket.Sockets.scc.SGO(socket.writeMsg, socket.Errorf)
	return socket
}

// Socket 基础网络连接
type Socket struct {
	storage.Data
	conn    net.Conn
	stop    chan struct{}
	Status  *Status
	Sockets *Sockets
	cwrite  chan *Message //写入通道,仅仅强制关闭的会被CLOSE
	netType NetType       //网络连接类型

}

func (this *Socket) emit(e EventType) bool {
	return this.Sockets.Emit(e, this)
}

func (this *Socket) close() {
	defer func() { _ = recover() }()
	if this.stop != nil {
		close(this.stop)
	}
	_ = this.conn.Close()
}

// destroy 彻底销毁,移除资源
func (this *Socket) destroy() {
	defer func() { _ = recover() }()
	this.Status.Destroy(func(r bool) {
		if !r {
			return
		}
		this.Sockets.Remove(this.Id())
		this.emit(EventTypeDestroyed)
	})
}

// disconnect 掉线,包含网络超时，网络错误
func (this *Socket) disconnect() {
	this.Status.Disconnect(func(r bool) {
		if !r {
			return
		}
		this.close()
		this.KeepAlive()
		this.emit(EventTypeDisconnect)
	})
}

// Close 强制关闭,无法重连
func (this *Socket) Close(msg ...*Message) {
	defer func() { _ = recover() }()
	this.Status.Close(func(r bool) {
		if !r {
			return
		}
		for _, m := range msg {
			_ = this.Write(m)
		}
	})
}

func (this *Socket) Errorf(format any) {
	this.Sockets.Errorf(this, format)
}

//func (this *Socket) Errorf(format any, args ...any) bool {
//	return this.Sockets.Errorf(this, format, args...)
//}

func (this *Socket) NetType() NetType {
	return this.netType
}

// Heartbeat 每一次Heartbeat() heartbeat计数加1
func (this *Socket) Heartbeat() {
	heartbeat := this.Status.Heartbeat()
	switch this.Status.status {
	case StatusTypeClosed:
		this.destroy()
	case StatusTypeDisconnect:
		if this.Get() == nil || Options.SocketReconnectTime == 0 || heartbeat > Options.SocketReconnectTime {
			this.destroy()
		}
	case StatusTypeClosing:
		if len(this.cwrite) == 0 || heartbeat >= Options.SocketDestroyingTime {
			this.close()
		}
	default:
		if heartbeat >= Options.SocketConnectTime {
			this.disconnect()
		}
	}
}

// KeepAlive 任何行为都清空heartbeat
func (this *Socket) KeepAlive() {
	this.Status.KeepAlive()
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
	if m == nil || this.cwrite == nil || !this.Status.Has(StatusTypeConnect, StatusTypeDisconnect) {
		return ErrSocketClosed
	}
	select {
	case this.cwrite <- m:
		return nil
	default:
		this.Close()
		return ErrSocketChannelFull
	}
}

func (this *Socket) processMsg(socket *Socket, msg *Message) {
	this.KeepAlive()
	this.Sockets.handle(socket, msg)
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
		this.Sockets.Errorf(this, "READ HEAD ERR,RemoteAddr:%v,HEAD:%v", err, this.RemoteAddr().String(), head)
		return false
	}
	//logger.Debug("READ HEAD:%+v BYTE:%v", *msg.Header, head)
	if msg.Len() > 0 {
		_, err = msg.Write(this.conn)
		if err != nil {
			this.Sockets.Errorf(this, "READ BODY ERR:%v", err)
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
	//logger.Debug("socket write error,code:%v, data:%v", msg.Code(), msg.Data())
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

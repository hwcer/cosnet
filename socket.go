package cosnet

import (
	"context"
	"github.com/hwcer/cosgo/storage"
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/logger"
	"io"
	"net"
	"sync/atomic"
)

const (
	StatusTypeConnect    = 0
	StatusTypeDisconnect = 1
	StatusTypeDestroying = 2
)

func NewSocket(engine *Agents, conn net.Conn, netType NetType) *Socket {
	socket := &Socket{conn: conn, Agents: engine, netType: netType}
	socket.stop = make(chan struct{})
	socket.cwrite = make(chan *Message, Options.WriteChanSize)
	socket.Agents.scc.CGO(socket.readMsg)
	socket.Agents.scc.CGO(socket.writeMsg)
	return socket
}

// Socket 基础网络连接
type Socket struct {
	storage.Data
	conn      net.Conn
	stop      chan struct{} //stop
	Agents    *Agents       //Agents
	cwrite    chan *Message //写入通道,仅仅强制关闭的会被CLOSE
	status    int32         //0-正常，1-断开，2-强制关闭
	netType   NetType       //网络连接类型
	heartbeat uint16        //heartbeat >=timeout 时被标记为超时
}

func (this *Socket) emit(e EventType) bool {
	return this.Agents.Emit(e, this)
}

// destroy 彻底销毁,移除资源
func (this *Socket) destroy() {
	if this.status != StatusTypeDestroying {
		atomic.StoreInt32(&this.status, StatusTypeDestroying)
	}
	if this.cwrite != nil {
		utils.Try(func() {
			close(this.cwrite)
		})
	}
	this.Agents.Remove(this.Id())
	this.emit(EventTypeDestroyed)
}

// disconnect 掉线,包含网络超时，网络错误
func (this *Socket) disconnect() {
	select {
	case <-this.stop:
	default:
		close(this.stop)
		this.emit(EventTypeDisconnect)
		_ = this.conn.Close()
		if atomic.CompareAndSwapInt32(&this.status, StatusTypeConnect, StatusTypeDisconnect) {
			this.KeepAlive()
		}
	}
}

// Close 强制关闭,无法重连
func (this *Socket) Close(msg ...*Message) {
	defer func() {
		_ = recover()
	}()
	for _, m := range msg {
		this.Write(m)
	}
	atomic.StoreInt32(&this.status, StatusTypeDestroying)
}

func (this *Socket) Errorf(format interface{}, args ...interface{}) bool {
	return this.Agents.Errorf(this, format, args...)
}

func (this *Socket) Status() int32 {
	return this.status
}
func (this *Socket) NetType() NetType {
	return this.netType
}

func (this *Socket) Player() *Player {
	v := this.Data.Get()
	if v == nil {
		return nil
	}
	p, _ := v.(*Player)
	return p
}

// Verified 是否身份认证
func (this *Socket) Verified() bool {
	return this.Get() != nil
}

// Heartbeat 每一次Heartbeat() heartbeat计数加1
func (this *Socket) Heartbeat() {
	this.heartbeat += Options.SocketHeartbeat
	switch this.status {
	case StatusTypeDisconnect:
		if !this.Verified() || Options.SocketReconnectTime == 0 || this.heartbeat > Options.SocketReconnectTime {
			this.destroy()
		}
	case StatusTypeDestroying:
		if len(this.cwrite) == 0 || this.heartbeat >= Options.SocketDestroyingTime {
			this.destroy()
		}
	default:
		if this.heartbeat >= Options.SocketConnectTime {
			this.disconnect()
		}
	}
}

// KeepAlive 任何行为都清空heartbeat
func (this *Socket) KeepAlive() {
	this.heartbeat = 0
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

// Write 外部写入消息
func (this *Socket) Write(m *Message) (re bool) {
	defer func() {
		if e := recover(); e != nil {
			re = this.Errorf(e)
		}
	}()
	if m == nil || this.cwrite == nil || this.status == StatusTypeDestroying {
		return false
	}
	select {
	case this.cwrite <- m:
		re = true
	default:
		logger.Debug("通道已满无法写消息:%v", this.Id())
		this.Close()
	}
	return
}

func (this *Socket) processMsg(socket *Socket, msg *Message) {
	this.KeepAlive()
	this.Agents.handle(socket, msg)
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
	msg := this.Agents.Acquire()
	defer func() {
		this.Agents.Release(msg)
	}()
	err := msg.Parse(head)
	if err != nil {
		logger.Debug("READ HEAD ERR,RemoteAddr:%v,HEAD:%v", err, this.RemoteAddr().String(), head)
		return this.Errorf(err)
	}
	//logger.Debug("READ HEAD:%+v BYTE:%v", *msg.Header, head)
	if msg.Len() > 0 {
		_, err = msg.Write(this.conn)
		if err != nil {
			logger.Debug("READ BODY ERR:%v", err)
			return
		}
	}
	this.processMsg(this, msg)
	return true
}

func (this *Socket) writeMsgTrue(msg *Message) (r bool) {
	defer func() {
		this.Agents.Release(msg)
	}()
	data, err := msg.Bytes()
	if err != nil {
		return this.Errorf(err)
	}
	//logger.Debug("socket write error,code:%v, data:%v", msg.Code(), msg.Data())
	var n int
	writeCount := 0
	for writeCount < len(data) {
		n, err = this.conn.Write(data[writeCount:])
		if err != nil {
			return this.Errorf(err)
		}
		writeCount += n
	}
	this.KeepAlive()
	return true
}

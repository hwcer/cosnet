package cosnet

import (
	"context"
	"encoding/json"
	"github.com/golang/protobuf/proto"
	"github.com/hwcer/cosgo/library/logger"
	"github.com/hwcer/cosgo/storage/cache"
	"io"
	"net"
	"sync/atomic"
)

const (
	StatusTypeNone    = 0
	StatusTypeOffline = 1
	StatusTypeRemoved = 2
)

//Socket 基础网络连接
type Socket struct {
	conn      net.Conn
	stop      chan struct{} //stop
	cwrite    chan *Message //写入通道,仅仅强制关闭的会被CLOSE
	status    int32         //0-正常，1-断开，2-强制关闭
	netType   NetType       //网络连接类型
	cosnet    *Cosnet
	heartbeat uint16 //heartbeat >=timeout 时被标记为超时
	cache.Data
}

func (this *Socket) emit(e EventType) {
	this.cosnet.Emit(e, this)
}

//start 启动工作进程
func (this *Socket) start() error {
	this.stop = make(chan struct{})
	this.cosnet.scc.CGO(this.readMsg)
	this.cosnet.scc.CGO(this.writeMsg)
	return nil
}

func (this *Socket) close() {
	select {
	case <-this.stop:
	default:
		close(this.stop)
		_ = this.conn.Close()
		if atomic.CompareAndSwapInt32(&this.status, StatusTypeNone, StatusTypeOffline) {
			this.KeepAlive()
		}
	}
}

//stopped 读写协程是否关闭或者正在关闭
//func (this *Socket) stopped() bool {
//	return this.alive
//}

//Close 强制关闭,无法重连
func (this *Socket) Close(msg ...*Message) {
	defer func() {
		_ = recover()
	}()
	for _, m := range msg {
		this.Write(m)
	}
	this.status = StatusTypeRemoved
	close(this.cwrite)
}

func (this *Socket) Status() int32 {
	return this.status
}
func (this *Socket) NetType() NetType {
	return this.netType
}

//Authenticated 是否身份认证
func (this *Socket) Authenticated() bool {
	return this.Get() != nil
}

//Heartbeat 每一次Heartbeat() heartbeat计数加1
func (this *Socket) Heartbeat() {
	this.heartbeat += Options.SocketHeartbeat
	switch this.status {
	case StatusTypeOffline:
		if Options.SocketReconnectTime == 0 || this.heartbeat > Options.SocketReconnectTime {
			this.cosnet.remove(this)
		}
	case StatusTypeRemoved:
		this.cosnet.remove(this)
	default:
		if this.heartbeat >= Options.SocketConnectTime {
			this.close()
		}
	}
}

//KeepAlive 任何行为都清空heartbeat
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

//Write 外部写入消息
func (this *Socket) Write(m *Message) (re bool) {
	defer func() {
		_ = recover()
	}()
	if this.status == StatusTypeRemoved || m == nil {
		return false
	}
	select {
	case this.cwrite <- m:
		re = true
	default:
		logger.Debug("通道已满无法写消息:%v", this.Id())
		this.close() //TODO 重试
	}
	return
}

//Json 发送Json数据
func (this *Socket) Json(code uint16, index uint16, data interface{}) (re bool) {
	msg := &Message{Header: NewHeader(code, index)}
	if data != nil {
		if err := json.NewEncoder(msg).Encode(data); err != nil {
			logger.Debug("socket Json error:%v", err)
			return
		}
	}
	return this.Write(msg)
}

//Protobuf 发送Protobuf
func (this *Socket) Protobuf(code uint16, index uint16, data proto.Message) (re bool) {
	msg := &Message{Header: NewHeader(code, index)}
	if msg != nil {
		var err error
		if msg.data, err = proto.Marshal(data); err != nil {
			logger.Debug("socket Protobuf error; Code:%v,err:%v", code, err)
			return false
		} else {
			msg.Header.size = int32(len(msg.data))
		}
	}
	return this.Write(msg)
}

func (this *Socket) processMsg(socket *Socket, msg *Message) {
	this.KeepAlive()
	//logger.Debug("processMsg:%+v", msg)
	err := this.cosnet.call(socket, msg)
	if err != nil {
		logger.Error("processMsg:%v", err)
		socket.close()
	}
}

func (this *Socket) readMsg(ctx context.Context) {
	defer this.close()
	var err error
	head := make([]byte, HeaderSize)
	for {
		_, err = io.ReadFull(this.conn, head)
		if err != nil {
			return
		}
		//logger.Debug("READ HEAD:%v", head)
		msg := &Message{Header: &Header{}}
		err = msg.Header.Parse(head)
		if err != nil {
			logger.Debug("READ HEAD ERR:%v", err)
			return
		}
		//logger.Debug("READ HEAD:%+v BYTE:%v", *msg.Header, head)
		if msg.Header.size > 0 {
			msg.data = make([]byte, msg.Header.size)
			_, err = io.ReadFull(this.conn, msg.data)
			if err != nil {
				logger.Debug("READ BODY ERR:%v", err)
				return
			}
		}
		this.processMsg(this, msg)
	}
}

func (this *Socket) writeMsg(ctx context.Context) {
	defer this.close()
	var msg *Message
	for {
		select {
		case <-ctx.Done():
			return //关闭服务器
		case <-this.stop:
			return //关闭SOCKET
		case msg = <-this.cwrite:
			if !this.writeMsgTrue(msg) {
				return
			}
		}
	}
}

func (this *Socket) writeMsgTrue(msg *Message) bool {
	if msg == nil {
		return false
	}
	data, err := msg.Bytes()
	if err != nil {
		return false
	}
	logger.Debug("write HEAD:%+v, DATA:%v", *msg.Header, data)
	var n int
	writeCount := 0
	for writeCount < len(data) {
		n, err = this.conn.Write(data[writeCount:])
		if err != nil {
			return false
		}
		writeCount += n
	}
	this.KeepAlive()
	return true
}

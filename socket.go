package cosnet

import (
	"context"
	"encoding/json"
	"github.com/golang/protobuf/proto"
	"github.com/hwcer/cosgo/library/logger"
	"github.com/hwcer/cosgo/storage/cache"
	"io"
	"net"
)

//Socket 基础网络连接
type Socket struct {
	conn      net.Conn
	stop      chan struct{} //stop
	cosnet    *Cosnet
	cwrite    chan *Message //写入通道
	netType   NetType       //网络连接类型
	stopping  int8          //正在关闭
	heartbeat uint16        //heartbeat >=timeout 时被标记为超时
	cache.Data
}

//start 启动工作进程，status启动前状态
func (this *Socket) start() error {
	this.stop = make(chan struct{})
	this.cosnet.scc.CGO(this.readMsg)
	this.cosnet.scc.CGO(this.writeMsg)
	return nil
}

//close 内部关闭方式
func (this *Socket) close() {
	select {
	case <-this.stop:
	default:
		close(this.stop)
	}
}

//stopped 读写协程是否关闭或者正在关闭
func (this *Socket) stopped() bool {
	select {
	case <-this.stop:
		return true
	default:
		return false
	}
}

//Close 强制关闭,无法重连
func (this *Socket) Close(msg ...*Message) {
	if len(msg) > 0 && !this.stopped() {
		this.stopping += 1
		for _, m := range msg {
			this.Write(m)
		}
	} else {
		this.close()
	}
}
func (this *Socket) NetType() NetType {
	return this.netType
}

//Heartbeat 每一次Heartbeat() heartbeat计数加1
func (this *Socket) Heartbeat() {
	this.heartbeat += 1
	if this.stopped() {
		this.cosnet.remove(this) //销毁
	} else if this.heartbeat >= Options.SocketConnectTime || (this.stopping > 0 && len(this.cwrite) == 0) {
		this.close()
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
	if m == nil {
		return false
	}
	if this.stopped() {
		//logger.Debug("SOCKET已经关闭无法写消息:%v", this.IId())
		return
	}
	select {
	case this.cwrite <- m:
		re = true
	default:
		logger.Debug(" 通道已满无法写消息:%v", this.Id())
		this.close()
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
	//if msg.Head != nil && msg.Head.Flags.Has(Message.FlagCompress) && msg.Data != nil {
	//	data, err := utils.GZipUnCompress(msg.Data)
	//	if err != nil {
	//		this.close()
	//		logger.Error("uncompress failed socket:%v err:%v", socket.IId(), err)
	//		return
	//	}
	//	msg.Data = data
	//	msg.Head.Flags.Leave(Message.FlagCompress)
	//	msg.Head.Size = uint32(len(msg.Data))
	//}
	//logger.Debug("processMsg:%+v", msg)
	this.cosnet.call(socket, msg)
}

func (this *Socket) readMsg(ctx context.Context) {
	defer this.close()
	var err error
	head := make([]byte, HeaderSize)
	for !this.stopped() {
		_, err = io.ReadFull(this.conn, head)
		if err != nil {
			return
		}
		//logger.Debug("READ HEAD:%v", head)
		msg := &Message{}
		err = msg.Header.Parse(head)
		if err != nil {
			//logger.Debug("READ ERR:%v", err)
			return
		}
		if msg.Header.size > 0 {
			msg.data = make([]byte, msg.Header.size)
			_, err = io.ReadFull(this.conn, msg.data)
			if err != nil {
				return
			}
		}
		this.processMsg(this, msg)
	}
}

func (this *Socket) writeMsg(ctx context.Context) {
	defer func() {
		if this.conn != nil {
			this.conn.Close()
		}
		this.close()
	}()
	var msg *Message
	for !this.stopped() {
		select {
		case <-this.stop:
			return
		case <-ctx.Done():
			return
		case msg = <-this.cwrite:
			if !this.writeMsgTrue(msg) {
				return
			}
		}
	}
}

func (this *Socket) writeMsgTrue(msg *Message) bool {
	if msg == nil {
		return true
	}
	data, err := msg.Bytes()
	if err != nil {
		return false
	}
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

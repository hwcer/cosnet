package cosnet

import (
	"context"
	"errors"
	"fmt"
	"github.com/hwcer/cosgo/logger"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosgo/session"
	"github.com/hwcer/cosgo/values"
	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/cosnet/message"
	"io"
	"net"
	"runtime/debug"
	"strconv"
	"sync/atomic"
)

func NewSocket(conn listener.Conn) *Socket {
	socket := &Socket{conn: conn}
	socket.id = atomic.AddUint64(&index, 1)
	socket.stop = make(chan struct{})
	socket.cwrite = make(chan message.Message, Options.WriteChanSize)
	scc.SGO(socket.readMsg)
	scc.SGO(socket.writeMsg)
	return socket
}

// Socket 基础网络连接
type Socket struct {
	id     uint64
	conn   listener.Conn
	data   *session.Data
	stop   chan struct{}
	cwrite chan message.Message //写入通道,仅仅强制关闭的会被CLOSE
	//status    Status
	closed    int32
	heartbeat uint32
}

func (sock *Socket) disconnect() bool {
	if !atomic.CompareAndSwapInt32(&sock.closed, 0, 1) {
		return false
	}
	defer func() {
		if err := recover(); err != nil {
			logger.Alert("Socket disconnect:%v", err)
		}
	}()
	close(sock.stop)
	_ = sock.conn.Close()
	sock.Emit(EventTypeDisconnect)
	sockets.Delete(sock.Id())
	sock.data = nil
	return true
}

func (sock *Socket) Id() uint64 {
	return sock.id
}

func (sock *Socket) Data() *session.Data {
	return sock.data
}
func (sock *Socket) Emit(e EventType) {
	Emit(e, sock)
}

// Close 强制关闭,无法重连
// todo 状态控制等待发送完再关闭
func (sock *Socket) Close(msg ...message.Message) {
	if sock.closed > 0 {
		return
	}
	defer func() {
		_ = recover()
	}()
	for _, m := range msg {
		_ = sock.Write(m)
	}
	if len(msg) > 0 {
		sock.KeepAlive()
	}
	sock.disconnect()
}

// OAuth 身份认证
func (sock *Socket) OAuth(v any, h ...func(*Socket)) {
	switch d := v.(type) {
	case map[string]interface{}:
		sock.data = session.NewData(strconv.FormatUint(sock.id, 10), "", d)
	case values.Values:
		sock.data = session.NewData(strconv.FormatUint(sock.id, 10), "", d)
	case *session.Data:
		sock.data = d
	default:
		logger.Error("unknown OAuth arg type:%v", v)
		return
	}
	sock.Emit(EventTypeAuthentication)
}

func (sock *Socket) Errorf(format any, args ...any) {
	Errorf(sock, format, args...)
}

// KeepAlive 任何行为都清空heartbeat
func (sock *Socket) KeepAlive() {
	sock.heartbeat = 0
}

func (sock *Socket) LocalAddr() net.Addr {
	if sock.conn != nil {
		return sock.conn.LocalAddr()
	}
	return nil
}
func (sock *Socket) RemoteAddr() net.Addr {
	if sock.conn != nil {
		return sock.conn.RemoteAddr()
	}
	return nil
}

func (sock *Socket) Send(path string, query values.Values, data any, async ...any) (err error) {
	m := message.Require()
	if err = m.Marshal(path, query, data); err != nil {
		return
	}
	return sock.Write(m, async...)
}

// Write 外部写入消息,慎用,注意发送失败时消息回收,参考Send
// async 异步发送
// 参数中如果有 func(socket *Socket) 类型，写入通道后 执行回调函数
func (sock *Socket) Write(m message.Message, async ...any) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	if sock.closed > 0 {
		return ErrSocketClosed
	}
	if len(async) == 0 {
		sock.cwrite <- m
	} else {
		go func() {
			sock.cwrite <- m
			for _, v := range async {
				if f, ok := v.(func(socket *Socket)); ok {
					f(sock)
				}
			}
		}()
	}
	return
}

func (sock *Socket) readMsg(_ context.Context) {
	defer sock.disconnect()
	for !scc.Stopped() {
		if msg, err := sock.conn.ReadMessage(); err != nil {
			if err != io.EOF && !errors.Is(err, net.ErrClosed) {
				sock.Errorf(err)
			}
			return
		} else if !sock.readMsgTrue(msg) {
			return
		}
	}
}
func (sock *Socket) readMsgTrue(msg message.Message) bool {
	if msg == nil {
		return true
	}
	defer message.Release(msg)
	if err := msg.Verify(); err != nil {
		sock.Errorf(err)
		return false
	}
	sock.KeepAlive()
	sock.handle(sock, msg)
	return true
}

func (sock *Socket) handle(socket *Socket, msg message.Message) {
	var err error
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("server handle error:%v\n%v", e, string(debug.Stack()))
		}
		if err != nil {
			Errorf(socket, err)
		}
	}()
	var path string
	if path, err = msg.Path(); err != nil {
		logger.Alert("Socket handle:%v", err)
		return
	}
	node, ok := Registry.Match(path)
	if !ok {
		Emit(EventTypeMessage, socket, msg)
		return
	}
	handler := node.Service.Handler.(*Handler)
	if handler == nil {
		return
	}
	c := &Context{Socket: socket, Message: msg}
	var reply interface{}
	reply, err = handler.Caller(node, c)
	if err == nil {
		err = handler.Serialize(c, reply)
	}
	if err != nil {
		sock.Errorf(err)
	}

}

func (sock *Socket) writeMsg(ctx context.Context) {
	defer sock.disconnect()
	for {
		select {
		case <-ctx.Done():
			return
		case <-sock.stop:
			return
		case msg := <-sock.cwrite:
			sock.writeMsgTrue(msg)
		}
	}
}

func (sock *Socket) writeMsgTrue(msg message.Message) {
	var err error
	defer func() {
		message.Release(msg)
		if err != nil {
			sock.Errorf(err)
		} else {
			sock.KeepAlive()
		}
	}()
	err = sock.conn.WriteMessage(msg)
}

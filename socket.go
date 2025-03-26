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
	id        uint64
	conn      listener.Conn
	data      *session.Data
	stop      chan struct{}
	magic     byte                 //使用的魔法数字
	cwrite    chan message.Message //写入通道,仅仅强制关闭的会被CLOSE
	closed    int32                // 1--正在关闭(等待通道中的消息全部发送完毕)  2-已经关闭
	heartbeat uint32
}

const (
	SocketClosedNone    int32 = 0
	SocketClosedClosing int32 = 1
	SocketClosedClosed  int32 = 2
)

func (sock *Socket) disconnect() bool {
	v := sock.closed
	if v == SocketClosedClosed {
		return true
	}
	if !atomic.CompareAndSwapInt32(&sock.closed, v, SocketClosedClosed) {
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
func (sock *Socket) Emit(e EventType, args ...any) {
	Emit(e, sock, args...)
}

// Close 强制关闭,无法重连
func (sock *Socket) Close() {
	atomic.CompareAndSwapInt32(&sock.closed, SocketClosedNone, SocketClosedClosing)
}

func (sock *Socket) setData(v any) {
	switch d := v.(type) {
	case map[string]any:
		sock.data = session.NewData(strconv.FormatUint(sock.id, 10), "", d)
	case values.Values:
		sock.data = session.NewData(strconv.FormatUint(sock.id, 10), "", d)
	case *session.Data:
		sock.data = d
	default:
		logger.Error("unknown OAuth arg type:%v", v)
		return
	}
}

// OAuth 身份认证
func (sock *Socket) OAuth(v any) {
	sock.setData(v)
	sock.Emit(EventTypeAuthentication)
}

// Replaced 被顶号
func (sock *Socket) Replaced(ip string) {
	sock.Emit(EventTypeReplaced, ip)
	sock.Close()
}

// Reconnect 断线重连,对于SOCKET而言本质上还是登陆，仅仅是事件不同，方便业务逻辑层面区分
func (sock *Socket) Reconnect(v any) {
	sock.setData(v)
	sock.Emit(EventTypeReconnected)
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
func (sock *Socket) Magic() byte {
	return sock.magic
}
func (sock *Socket) Send(index int32, path string, data any) (err error) {
	m := message.Require()
	if err = m.Marshal(sock.magic, index, path, data); err != nil {
		return
	}
	return sock.Write(m)
}

// Async 异步写入数据
func (sock *Socket) Async(m message.Message) (s chan struct{}, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	if sock.closed > 0 {
		return s, ErrSocketClosed
	}
	s = make(chan struct{})
	go func() {
		sock.cwrite <- m
		close(s)
	}()
	return s, nil
}

// Write 外部写入消息,慎用,注意发送失败时消息回收,参考Send
// 参数中如果有 func(socket *Socket) 类型，写入通道后 执行回调函数
func (sock *Socket) Write(m message.Message) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	if sock.closed > 0 {
		return ErrSocketClosed
	}
	sock.cwrite <- m
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
	sock.KeepAlive()
	if sock.magic == 0 {
		magic := msg.Magic()
		sock.magic = magic.Key
	}
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
	if path, _, err = msg.Path(); err != nil {
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
	if err = handler.Caller(node, c); err != nil {
		socket.Errorf(err)
	}
	//if msg.Confirm() {
	//	err = c.Reply(reply)
	//}
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
			//继续读取数据
			for i := 0; i < len(sock.cwrite); i++ {
				if msg = <-sock.cwrite; msg != nil {
					sock.writeMsgTrue(msg)
				}
			}
			//如果已经准备关闭，就退出循环
			if sock.closed == SocketClosedClosing {
				return
			}
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

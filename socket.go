package cosnet

import (
	"context"
	"errors"
	"io"
	"net"
	"runtime/debug"
	"sync/atomic"

	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosgo/session"
	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/cosnet/message"
	"github.com/hwcer/logger"
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
	data      *session.Data        //登录后绑定的用户信息
	stop      chan struct{}        //关闭标记
	magic     byte                 //使用的魔法数字
	cwrite    chan message.Message //写入通道,仅仅强制关闭的会被CLOSE
	status    int32                // 1--正在关闭(等待通道中的消息全部发送完毕)  2-已经关闭
	heartbeat int32
}

const (
	SocketStatusNone    int32 = 0
	SocketStatusClosing int32 = 1
	SocketStatusClosed  int32 = 2
)

func (sock *Socket) disconnect() bool {
	v := sock.status
	if v == SocketStatusClosed {
		return true
	}
	if !atomic.CompareAndSwapInt32(&sock.status, v, SocketStatusClosed) {
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
	sockets.Delete(sock.id)
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
// delay 延时关闭，单位秒
func (sock *Socket) Close(delay ...int32) {
	if !atomic.CompareAndSwapInt32(&sock.status, SocketStatusNone, SocketStatusClosing) {
		return
	}
	sock.heartbeat = Options.SocketConnectTime
	if len(delay) > 0 {
		sock.heartbeat -= delay[0]
	}
}

// Authentication 身份认证
func (sock *Socket) Authentication(v *session.Data, reconnect ...bool) {
	sock.data = v
	var r bool
	if len(reconnect) > 0 {
		r = reconnect[0]
	}
	sock.Emit(EventTypeAuthentication, r)
	if r {
		sock.Emit(EventTypeReconnected)
	}
}

// Replaced 被顶号
func (sock *Socket) Replaced(ip string) {
	sock.Emit(EventTypeReplaced, ip)
	sock.data = nil //取消与角色关联，避免触发角色的掉线事件
	sock.Close(Options.SocketReplacedTime)
}

func (sock *Socket) Errorf(format any, args ...any) {
	Errorf(sock, format, args...)
}

// KeepAlive 任何行为都清空heartbeat
func (sock *Socket) KeepAlive() {
	if sock.status == SocketStatusNone {
		sock.heartbeat = 0
	}
	if sock.data != nil {
		sock.data.KeepAlive()
	}
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

func (sock *Socket) Send(index int32, path string, data any) {
	m := message.Require()
	if err := m.Marshal(sock.magic, index, path, data); err == nil {
		sock.Write(m)
	} else {
		logger.Error(err)
	}
}

// Async 异步写入数据
func (sock *Socket) Async(m message.Message) (s chan struct{}) {
	defer func() {
		if e := recover(); e != nil {
			logger.Error(e)
		}
	}()
	s = make(chan struct{})
	go func() {
		defer func() {
			close(s)
		}()
		if sock.Alive() {
			sock.cwrite <- m
		}
	}()
	return
}

// Write 外部写入消息,慎用,注意发送失败时消息回收,参考Send
// 参数中如果有 func(socket *Socket) 类型，写入通道后 执行回调函数
func (sock *Socket) Write(m message.Message) {
	defer func() {
		if e := recover(); e != nil {
			logger.Error(e)
		}
	}()
	if sock.Alive() {
		sock.cwrite <- m
	}
}

func (sock *Socket) Alive() bool {
	return sock.status == SocketStatusNone && sock.conn != nil
}

func (sock *Socket) Heartbeat(v int32) int32 {
	sock.heartbeat += v
	return sock.heartbeat
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
	defer func() {
		if e := recover(); e != nil {
			socket.Errorf("server handle error:%v\n%v", e, string(debug.Stack()))
		}
	}()
	path, _, err := msg.Path()
	if err != nil {
		socket.Errorf("message path error code:%d error:%v", msg.Code(), err)
		return
	}
	node, ok := Registry.Match(RegistryMethod, path)
	if !ok {
		socket.Emit(EventTypeMessage, msg)
		return
	}
	handler := node.Handler().(*Handler)
	if handler == nil {
		socket.Errorf("no handler for %s", path)
		return
	}
	c := &Context{Socket: socket, Message: msg}
	reply := handler.handle(node, c)
	if err = handler.write(c, reply); err != nil {
		socket.Errorf("write reply message error,path:%s,errMsg:%v", path, err)
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
	defer func() {
		message.Release(msg)
		if e := recover(); e != nil {
			sock.Errorf(e)
		}
	}()
	if err := sock.conn.WriteMessage(msg); err != nil {
		sock.Errorf(err)
	}
}

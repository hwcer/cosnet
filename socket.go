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

// Socket 基础网络连接
type Socket struct {
	id        uint64
	conn      listener.Conn
	data      *session.Data        //登录后绑定的用户信息
	stop      chan struct{}        //关闭标记
	magic     byte                 //使用的魔法数字
	cwrite    chan message.Message //写入通道,仅仅强制关闭的会被CLOSE
	status    int32                // 1--正在关闭(等待通道中的消息全部发送完毕)  2-已经关闭
	cosnet    *NetHub
	address   *string //作为客户端时，连接的服务器地址,为空时被视为服务器端socket
	heartbeat int32
}

const (
	SocketStatusNone    int32 = 0
	SocketStatusClosing int32 = 1
	SocketStatusClosed  int32 = 2
)

func (sock *Socket) connect(conn listener.Conn) {
	sock.conn = conn
	sock.stop = make(chan struct{})
	scc.SGO(sock.readMsg)
	scc.SGO(sock.writeMsg)
}
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
	sock.conn = nil
	if sock.address != nil {
		return sock.reconnect() //作为客户端，不触发事件直接尝试重连
	}
	sock.Emit(EventTypeDisconnect)
	sock.cosnet.sockets.Delete(sock.id)
	sock.data = nil

	// 释放通道中的所有消息
	for {
		select {
		case msg, ok := <-sock.cwrite:
			if !ok {
				return true
			}
			message.Release(msg)
		default:
			return true
		}
	}
}

// reconnect 断线重连，仅仅作为客户端时自动重连服务器
func (sock *Socket) reconnect() bool {
	address := *sock.address
	logger.Alert("socket reconnect:%s", address)
	scc.SGO(func(ctx context.Context) {
		for !scc.Stopped() {
			if conn, err := sock.cosnet.tryConnect(address); err == nil {
				sock.connect(conn)
				return
			}
		}
	})
	return false
}

func (sock *Socket) Id() uint64 {
	return sock.id
}

func (sock *Socket) Data() *session.Data {
	return sock.data
}

func (sock *Socket) Emit(e EventType, args ...any) {
	sock.cosnet.Emit(e, sock, args...)
}
func (sock *Socket) Conn() listener.Conn {
	return sock.conn
}
func (sock *Socket) Type() listener.SocketType {
	if sock.address != nil {
		return listener.SocketTypeClient
	}
	return listener.SocketTypeServer
}

func (sock *Socket) NetHub() *NetHub {
	return sock.cosnet
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
	sock.cosnet.Errorf(sock, format, args...)
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

func (sock *Socket) Send(flag message.Flag, index int32, path string, data any) {
	m := message.Require()
	magic := sock.magic
	if magic == 0 {
		magic = message.MagicNumberPathJson
	}
	if err := m.Marshal(magic, flag, index, path, data); err == nil {
		sock.Write(m)
	} else {
		message.Release(m)
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

func (sock *Socket) readMsg(_ context.Context) {
	defer sock.disconnect()
	for !scc.Stopped() {
		msg := message.Require()
		if err := sock.conn.ReadMessage(sock, msg); err != nil {
			message.Release(msg)
			if err != io.EOF && !errors.Is(err, net.ErrClosed) {
				sock.Errorf(err)
			}
			return
		}
		sock.readMsgTrue(msg)
		message.Release(msg)
	}
}

func (sock *Socket) readMsgTrue(msg message.Message) {
	sock.KeepAlive()
	magic := msg.Magic()
	if magic == nil || magic.Key == 0 {
		logger.Debug("magic is nil :%v", msg)
		return //未被初始化的消息
	}
	if sock.magic == 0 {
		sock.magic = magic.Key
	}
	sock.handle(sock, msg)
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
	node, _ := sock.cosnet.Registry.Search(RegistryMethod, path)
	if node == nil {
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
	if err = handler.reply(c, reply); err != nil {
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
	if err := sock.conn.WriteMessage(sock, msg); err != nil {
		sock.Errorf(err)
	}
}

// doHeartbeat 执行单个连接的心跳检查
// 每一次Heartbeat()调用，heartbeat计数加1
// 参数:
//
//	v: 心跳计数增量
func (sock *Socket) Heartbeat(v int32) int32 {
	// 如果设置了连接超时时间，并且心跳计数超过了超时时间，则断开连接
	sock.heartbeat += v
	if Options.SocketConnectTime > 0 && sock.heartbeat > Options.SocketConnectTime {
		sock.disconnect()
	}
	return sock.heartbeat
}

//func (sock *Socket) Heartbeat(v int32) int32 {
//	sock.heartbeat += v
//	return sock.heartbeat
//}

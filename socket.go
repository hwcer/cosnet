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

// Socket 表示一个网络连接，封装了底层的网络连接和会话数据。
type Socket struct {
	id        uint64               // 唯一标识符
	conn      listener.Conn        // 底层网络连接
	data      *session.Data        // 登录后绑定的用户会话数据
	stop      chan struct{}        // 关闭信号通道
	magic     byte                 // 消息魔数，用于消息格式识别
	cwrite    chan message.Message // 写入通道，用于异步发送消息
	status    int32                // 连接状态：0-正常，1-正在关闭，2-已关闭
	sockets   *Sockets             // 所属的 Sockets 管理器
	address   string               // 客户端模式：连接的服务器地址,为空时代表是服务器模式
	heartbeat int32                // 心跳计数器
}

// Socket 状态常量。
const (
	SocketStatusNone    int32 = 0 // 正常状态
	SocketStatusClosing int32 = 1 // 正在关闭（等待通道中的消息发送完毕）
	SocketStatusClosed  int32 = 2 // 已关闭
)

func (sock *Socket) connect(conn listener.Conn) {
	sock.conn = conn
	sock.stop = make(chan struct{})
	sock.status = SocketStatusNone
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
	sock.Emit(EventTypeDisconnect)
	if sock.address != "" {
		return sock.tryReconnect() //作为客户端，不触发事件直接尝试重连
	}
	sock.release()
	return true
}
func (sock *Socket) release() {
	sock.sockets.sockets.Delete(sock.id)
	atomic.AddInt64(&sock.sockets.count, -1)
	sock.data = nil
	// 释放通道中的所有消息
	for {
		select {
		case msg, ok := <-sock.cwrite:
			if !ok {
				return
			}
			message.Release(msg)
		default:
			return
		}
	}
}

// reconnect 断线重连，仅仅作为客户端时自动重连服务器
func (sock *Socket) tryReconnect() bool {
	address := sock.address
	logger.Alert("socket reconnect:%s", address)
	scc.SGO(func(ctx context.Context) {
		if conn, err := sock.sockets.tryConnect(ctx, address, 0); err == nil {
			sock.connect(conn)
			return
		}
		sock.release()
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
	sock.sockets.Emit(e, sock, args...)
}
func (sock *Socket) Conn() listener.Conn {
	return sock.conn
}
func (sock *Socket) Type() listener.SocketType {
	if sock.address != "" {
		return listener.SocketTypeClient
	}
	return listener.SocketTypeServer
}

func (sock *Socket) Sockets() *Sockets {
	return sock.sockets
}

// Close 强制关闭 Socket，关闭后不会自动重连。
// 参数 delay: 延时关闭时间（秒），可选。
func (sock *Socket) Close(delay ...int32) {
	if !atomic.CompareAndSwapInt32(&sock.status, SocketStatusNone, SocketStatusClosing) {
		return
	}
	sock.heartbeat = Options.SocketConnectTime
	if len(delay) > 0 {
		sock.heartbeat -= delay[0]
	}
}

// Authentication 进行身份认证，绑定用户会话数据。
// 参数:
//   - v: 用户会话数据
//   - reconnect: 是否为重连，可选
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

// Replaced 处理被顶号（同一账号在其他地方登录）。
// 参数 ip: 新登录的 IP 地址。
func (sock *Socket) Replaced(ip string) {
	sock.Emit(EventTypeReplaced, ip)
	sock.data = nil // 取消与角色关联，避免触发角色的掉线事件
	sock.Close(Options.SocketReplacedTime)
}

func (sock *Socket) Errorf(format any, args ...any) {
	sock.sockets.Errorf(sock, format, args...)
}

// KeepAlive 重置心跳计数器，表示连接活跃。
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

// Async 异步写入消息到发送通道。
// 参数 m: 要发送的消息。
// 返回值: 发送完成信号通道，关闭表示消息已加入发送队列。
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
		if sock.IsReady() {
			sock.cwrite <- m
		}
	}()
	return
}

// Write 将消息写入发送通道。
// 参数 m: 要发送的消息。
// 注意: 慎用，注意发送失败时消息回收，参考 Send 方法。
func (sock *Socket) Write(m message.Message) {
	defer func() {
		if e := recover(); e != nil {
			logger.Error(e)
		}
	}()
	if sock.IsReady() {
		sock.cwrite <- m
	}
}

// IsReady 检查 Socket 是否处于可读写状态。
// 返回值: 如果 Socket 状态正常且已连接则返回 true。
func (sock *Socket) IsReady() bool {
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
	node, _ := sock.sockets.Registry.Search(RegistryMethod, path)
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

// Heartbeat 执行心跳检测，检查连接是否超时。
// 参数 v: 心跳计数增量。
// 返回值: 当前心跳计数。
func (sock *Socket) Heartbeat(v int32) int32 {
	// 如果设置了连接超时时间，并且心跳计数超过了超时时间，则断开连接
	sock.heartbeat += v
	if Options.SocketConnectTime > 0 && sock.heartbeat > Options.SocketConnectTime {
		sock.disconnect()
	} else {
		sock.Emit(EventTypeHeartbeat, v)
	}
	return sock.heartbeat
}

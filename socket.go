package cosnet

import (
	"context"
	"errors"
	"fmt"
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

type SocketAsyncResult struct {
	e error
	c chan struct{}
}

func (r *SocketAsyncResult) Wait() error {
	<-r.c
	return r.e
}

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

	//有效状态

	SocketStatusNone      int32 = iota //初始状态，工作协程未启动,处理完初始化程序后转换为 SocketStatusConnected
	SocketStatusConnected              //SOCKET 已连接，可以正常工作,断开连接时 转换为 SocketStatusDisconnect
	SocketStatusClosing                // 正在手动关闭（等待通道中的消息发送完毕）

	// 无效 或者过度状态

	SocketStatusDisconnect   //断开连接时，仅仅在disconnect 方法内部使用,根据情况直接转换为 SocketStatusDisconnected 或者 SocketStatusReconnecting
	SocketStatusDisconnected //断开连接后，自动销毁
	SocketStatusReconnecting //断线重连中（客户端）,连接失败时会销毁,成功后转为 SocketStatusConnected
	SocketStatusReleased     //已关闭，无法再复活
)

// connect 处理连接成功后的一些状态
// 仅仅在Create 和 tryReconnect 中调用，可以安全的对 status 赋值
func (sock *Socket) connect(conn listener.Conn) {
	sock.conn = conn
	sock.stop = make(chan struct{})
	sock.status = SocketStatusConnected
	sock.heartbeat = 0
	sock.Emit(EventTypeConnected)
	scc.SGO(sock.readMsg)
	scc.SGO(sock.writeMsg)
}

// isValidStatus 检查状态是否为活跃状态（可以执行操作的状态）
func isValidStatus(status int32) bool {
	return status == SocketStatusNone || status == SocketStatusConnected || status == SocketStatusClosing
}

// disconnect 断开连接时
// 在工作协程和心跳中调用，仅仅当 SocketStatusConnected 时可以使用
func (sock *Socket) disconnect() bool {
	status := sock.status
	if !isValidStatus(status) {
		return false
	}
	if !atomic.CompareAndSwapInt32(&sock.status, status, SocketStatusDisconnect) {
		return false
	}
	defer func() {
		if err := recover(); err != nil {
			logger.Alert("Socket disconnect:%v", err)
		}
	}()
	close(sock.stop)
	if sock.conn != nil {
		_ = sock.conn.Close()
		sock.conn = nil
	}
	sock.Emit(EventTypeDisconnect)
	if sock.Type() == listener.SocketTypeClient {
		sock.status = SocketStatusReconnecting
		return sock.tryReconnect()
	}
	sock.status = SocketStatusDisconnected
	sock.release()
	return true
}

// release 销毁socket
func (sock *Socket) release() {
	sock.status = SocketStatusReleased
	atomic.AddInt64(&sock.sockets.count, -1)
	sock.sockets.sockets.Delete(sock.id)
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
	if !atomic.CompareAndSwapInt32(&sock.status, SocketStatusConnected, SocketStatusClosing) {
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
// 仅在 SocketStatusNone 或 SocketStatusConnected 状态下有效。
func (sock *Socket) KeepAlive() {
	if sock.status == SocketStatusConnected {
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

func (sock *Socket) Send(flag message.Flag, index int32, path string, data any) error {
	m := message.Require()
	magic := sock.magic
	if magic == 0 {
		magic = message.MagicNumberPathJson
	}
	if err := m.Marshal(magic, flag, index, path, data); err != nil {
		message.Release(m)
		return fmt.Errorf("socket send marshal error: %w", err)
	}
	if err := sock.Write(m); err != nil {
		return fmt.Errorf("socket send write error: %w", err)
	}
	return nil
}

// Async 异步写入消息到发送通道。
// 参数 m: 要发送的消息。
// 返回值: 发送完成信号通道，关闭表示消息已加入发送队列。
func (sock *Socket) Async(m message.Message) (r *SocketAsyncResult) {
	r = &SocketAsyncResult{
		c: make(chan struct{}),
	}
	go func() {
		defer func() {
			if e := recover(); e != nil {
				r.e = fmt.Errorf("socket async panic: %v", e)
			}
			close(r.c)
		}()
		if !sock.IsReady() {
			r.e = fmt.Errorf("socket not ready, status: %d", sock.status)
			return
		}
		sock.cwrite <- m
	}()
	return
}

// Write 将消息写入发送通道。
// 参数 m: 要发送的消息。
// 返回值: 错误信息，如果 Socket 未就绪则返回错误。
// 注意: 慎用，注意发送失败时消息回收，参考 Send 方法。
func (sock *Socket) Write(m message.Message) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("socket write panic: %v", e)
		}
	}()
	if !sock.IsReady() {
		return fmt.Errorf("socket not ready, status: %d", sock.status)
	}
	sock.cwrite <- m
	return nil
}

// IsReady 检查 Socket 是否处于可读写状态。
// 返回值: 如果 Socket 状态正常且已连接则返回 true。
func (sock *Socket) IsReady() bool {
	return sock.status == SocketStatusConnected
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
	status := sock.status
	if !isValidStatus(status) {
		return sock.heartbeat
	}
	sock.heartbeat += v
	if Options.SocketConnectTime > 0 && sock.heartbeat > Options.SocketConnectTime {
		sock.disconnect()
	} else {
		sock.Emit(EventTypeHeartbeat, v)
	}
	return sock.heartbeat
}

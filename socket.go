package cosnet

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

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
	//主动关闭(Close 置的 SocketStatusClosing)或已进入退出流程时不再重连,否则进程退不掉:
	//scc 取消后 readMsg/writeMsg 会立刻返回并在 defer 里 disconnect,此时状态仍是
	//SocketStatusConnected,只判 Closing 会重连成功->工作协程又立刻退出->再重连,死循环
	if sock.Type() == listener.SocketTypeClient && status != SocketStatusClosing && !sock.sockets.stopped() {
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
	if sock.sockets.stopped() {
		sock.release()
		return false
	}
	address := sock.address
	logger.Alert("socket reconnect:%s", address)
	scc.SGO(func(ctx context.Context) {
		//首拨也要等一拍。tryConnect 的退避是按"连不上"设计的(第一次立即拨),
		//但对端"连上即断"(端口被别的服务占用、协议不匹配)时,重连链路上没有任何等待,
		//会变成毫秒级死循环:拨通->工作协程立刻退出->再拨通,日志刷屏且空转 CPU
		if !sock.waitReconnect(ctx) {
			sock.release()
			return
		}
		conn, err := sock.sockets.tryConnect(ctx, address, 0)
		if err != nil {
			sock.release()
			return
		}
		//tryConnect 是先拨号后查 ctx,拨号期间进入退出流程时会带回一条可用连接,
		//这里必须丢掉:再 connect 就等于把读写协程重新拉起来,关闭流程永远收不了尾
		if sock.sockets.stopped() {
			_ = conn.Close()
			sock.release()
			return
		}
		sock.connect(conn)
	})
	return false
}

// waitReconnect 重连前的等待，返回 false 表示等待期间已进入退出流程
func (sock *Socket) waitReconnect(ctx context.Context) bool {
	delay := sock.sockets.Options.ClientReconnectTime
	if delay <= 0 {
		return !sock.sockets.stopped()
	}
	timer := time.NewTimer(time.Duration(delay) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return !sock.sockets.stopped()
	}
}

func (sock *Socket) Id() uint64 {
	return sock.id
}
func (sock *Socket) Is(target *Socket) bool {
	if target == nil {
		return false
	}
	return sock.id == target.id
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

// Close 关闭 Socket，关闭后不会自动重连。
// 参数 delay: 期望的剩余存活秒数（可选，0=尽快关闭）。
// 返回值: 是否由本次调用切入 SocketStatusClosing（已在关闭流程中返回 false）。
//
// 关闭是"预约"而非立即执行：只切状态并把心跳计数器推到目标位置，真正断开由
// Heartbeat 走到 SocketConnectTime 时触发。期间连接**仍可写**（见 CanWrite），
// 在途回包能发完，这正是顶号协商期赖以工作的机制。
//
// ⚠️ 只缩短剩余存活时间，不延长：已经失联一阵(heartbeat 累积很高)的连接不能靠
// Close 续命——否则对一条本来还剩 5 秒就会被判死的半开连接发起顶号协商，
// 反而让它多活一整个协商期，新端要等更久。
func (sock *Socket) Close(delay ...int32) bool {
	if !atomic.CompareAndSwapInt32(&sock.status, SocketStatusConnected, SocketStatusClosing) {
		return false
	}
	h := Options.SocketConnectTime
	if len(delay) > 0 {
		h -= delay[0]
	}
	if h > sock.heartbeat {
		sock.heartbeat = h
	}
	return true
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

// Replaced 收到顶号请求：**不立即踢掉**，而是进入顶号协商期。
// 参数 ip: 请求顶号的客户端 IP。
// 返回值: 是否由本次调用发起协商（已在协商期中返回 false，不重复通知、不重置倒计时）。
//
// 协商期内这条连接 **只收不发**（CanWrite 仍为 true、CanRead 已为 false）：
// 在途回包与服务器推送照常送达，客户端能把手头的事看完；它自己发来的新请求
// 一律被拒。倒计时结束由心跳断开，届时会话上的 socket 被清空，新端再登录即可上线。
//
// ⚠️ 先 Close 再 Emit：通知包要带上准确的剩余秒数。旧实现反过来（先 Emit 后 Close），
// 那条通知能发出去纯属侥幸——Close 之后 Write 就被挡住了，顺序一换就静默丢包。
// 现在 CanWrite 放开了 Closing，这个顺序才是安全的。
//
// ⚠️ 不再清空 sock.data：协商期内还要靠它定位会话推消息；更关键的是倒计时结束
// 断开时，玩家此刻是**真的离线**（新端还没进来），data 为 nil 会让 EventSessionDisconnect
// 整个丢掉。旧实现清它，是因为旧流程里新连接已经立刻接管、旧连接断开不算掉线。
func (sock *Socket) Replaced(ip string) bool {
	if !sock.Close(Options.SocketReplacedTime) {
		return false
	}
	sock.Emit(EventTypeReplaced, &Replaced{Address: ip, Timeout: sock.Countdown()})
	return true
}

func (sock *Socket) Errorf(format any, args ...any) {
	sock.sockets.Errorf(sock, format, args...)
}

// KeepAlive 重置心跳计数器，表示连接活跃。
// 仅在 SocketStatusConnected 状态下重置 socket 自身的计数器——
// ⚠️ 关闭流程(含顶号协商期)故意不重置：否则被顶号的一方只要不停发包就能无限续命，
// 倒计时永远走不完，新端也就永远上不来。
// 会话心跳(data.KeepAlive)不受此限:协商期内会话必须保活,不能让它先于连接过期。
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

// Magic 设置或获取 Socket 的魔数。
func (sock *Socket) Magic(magic ...byte) byte {
	if len(magic) > 0 {
		sock.magic = magic[0]
	}
	return sock.magic
}

func (sock *Socket) Send(flag message.Flag, index int32, path any, data any, safe ...bool) error {
	magic := sock.magic
	if magic == 0 {
		magic = message.Options.Magic
	}
	return sock.SendWithMagic(magic, flag, index, path, data, safe...)
}

func (sock *Socket) SendWithMagic(magic byte, flag message.Flag, index int32, path any, data any, safe ...bool) error {
	m := message.Require()
	if err := m.Marshal(magic, flag, index, path, data); err != nil {
		message.Release(m)
		return fmt.Errorf("socket send marshal error: %w", err)
	}
	//logger.Debug("SendWithMagic:%d index:%d path:%s", magic, index, path)
	if err := sock.Write(m, safe...); err != nil {
		message.Release(m)
		return fmt.Errorf("socket send write error: %w", err)
	}
	return nil
}

// Async 异步写入消息到发送通道。
// 参数:
//   - m: 要发送的消息
//   - safe: 可选，安全模式（默认 true），为 false 时通道满则丢弃消息
//
// 返回值: 发送完成信号通道，关闭表示消息已加入发送队列或发生错误。
func (sock *Socket) Async(m message.Message, safe ...bool) (r *SocketAsyncResult) {
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
		r.e = sock.Write(m, safe...)
	}()
	return
}

// Write 将消息写入发送通道。
// 参数 m: 要发送的消息。
// 返回值: 错误信息，如果 Socket 未就绪或通道已满则返回错误。
// 注意: 慎用，注意发送失败时消息回收，参考 Send 方法。
func (sock *Socket) Write(m message.Message, safe ...bool) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("socket write panic: %v", e)
		}
	}()
	if !sock.CanWrite() {
		return fmt.Errorf("socket not ready, status: %d", sock.status)
	}
	// safe 模式（默认）：阻塞等待通道可用，但监听 stop 避免 socket 关闭后永久阻塞
	// 非 safe 模式：通道满时直接丢弃
	if len(safe) > 0 && !safe[0] {
		select {
		case sock.cwrite <- m:
		case <-sock.stop:
			return fmt.Errorf("socket closed")
		default:
			return fmt.Errorf("socket write channel full")
		}
	} else {
		select {
		case sock.cwrite <- m:
		case <-sock.stop:
			return fmt.Errorf("socket closed")
		}
	}
	return nil
}

// IsReady 检查 Socket 是否完全正常（已连接且不在关闭流程中）。
// 判断"能不能收请求/能不能发消息"请用 CanRead / CanWrite——关闭流程中两者并不同步。
func (sock *Socket) IsReady() bool {
	return sock.status == SocketStatusConnected
}

// CanRead 是否受理客户端发来的请求（入站）。
// 顶号协商期(SocketStatusClosing)为 false：被顶号的一方**只收不发**，
// 它的新请求会被直接拒掉，不再进业务层。
func (sock *Socket) CanRead() bool {
	return sock.status == SocketStatusConnected
}

// CanWrite 是否可向客户端发送消息（出站）。
// 顶号协商期(SocketStatusClosing)仍为 true：在途回包、服务器推送、顶号通知都要发得出去。
// ⚠️ 这里放开 Closing 是整个协商流程的前提。收紧成 IsReady 的话，Close 一执行
// 后续所有 Write 就全返回 "socket not ready"，而 deliver / handler.reply 里都是
// `_ = sock.Send(...)`，错误被吞——线上表现为"服务端一切正常、客户端什么都收不到"。
func (sock *Socket) CanWrite() bool {
	return sock.status == SocketStatusConnected || sock.status == SocketStatusClosing
}

// Countdown 关闭倒计时剩余秒数；不在关闭流程中返回 0。
// 精度受心跳 tick 间隔限制(Options.Heartbeat，默认 10 秒)，只用于提示，别拿它做精确判定。
func (sock *Socket) Countdown() int32 {
	if sock.status != SocketStatusClosing {
		return 0
	}
	if r := Options.SocketConnectTime - sock.heartbeat; r > 0 {
		return r
	}
	return 0
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
	sock.magic = magic.Key
	sock.handle(msg)
}

func (sock *Socket) handle(msg message.Message) {
	defer func() {
		if e := recover(); e != nil {
			sock.Errorf("server handle error:%v", e)
		}
	}()
	path, _, err := msg.Path()
	if err != nil {
		sock.Errorf("message path error code:%d error:%v", msg.Code(), err)
		return
	}

	node, _ := sock.sockets.Registry.Search(RegistryMethod, path)
	if node == nil {
		if !sock.CanRead() {
			return //协商期拒收:未注册的消息没有 handler 可以回，只能丢弃
		}
		sock.Emit(EventTypeMessage, msg)
		return
	}
	handler := node.Handler().(*Handler)
	if handler == nil {
		sock.Errorf("no handler for %s", path)
		return
	}
	c := &Context{Socket: sock, Message: msg}
	var reply any
	if sock.CanRead() {
		reply = handler.handle(node, c)
	} else {
		//顶号协商期:只收不发。仍然走 handler.reply,回包才会过业务层的 Serialize
		//被包成正常的确认包结构——直接丢弃的话客户端只能挂到超时,而且报的是笼统的网络错误
		reply = session.ErrorSessionReplaced
	}
	if err = handler.reply(c, reply); err != nil {
		sock.Errorf("write reply message error,path:%s,errMsg:%v", path, err)
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
		if e := recover(); e != nil {
			sock.Errorf(e)
		}
		message.Release(msg)
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

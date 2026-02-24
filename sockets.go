package cosnet

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hwcer/cosgo"
	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/cosgo/values"
	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/cosnet/message"
	"github.com/hwcer/cosnet/tcp"
	"github.com/hwcer/cosnet/udp"
	"github.com/hwcer/cosnet/wss"
	"github.com/hwcer/logger"
	"golang.org/x/sync/syncmap"
)

// Matcher 检查输入流是否包含有效的消息魔术数字。
// 参数 r: 输入流读取器。
// 返回值: 是否包含有效的消息魔术数字。
func Matcher(r io.Reader) bool {
	buf := make([]byte, 1)
	n, _ := r.Read(buf)
	return n == 1 && message.Magics.Has(buf[0])
}

// New 创建一个新的 Sockets 管理器。
// 参数 heartbeat: 心跳间隔（秒）。
// 返回值: Sockets 管理器实例。
func New(heartbeat int32) *Sockets {
	ss := &Sockets{
		index:     0,
		sockets:   syncmap.Map{},
		emitter:   make(map[EventType][]EventsFunc),
		instance:  make([]listener.Listener, 0),
		heartbeat: heartbeat,
		Registry:  registry.New(),
	}
	return ss
}

// Sockets 管理 Socket 连接的集合，包含服务器和客户端功能。
type Sockets struct {
	index     uint64                     // Socket 索引计数器
	started   atomic.Bool                // 是否已启动
	sockets   syncmap.Map                // 存储所有 Socket 连接
	emitter   map[EventType][]EventsFunc // 事件监听器映射
	instance  []listener.Listener        // 监听器实例列表
	heartbeat int32                      // 心跳间隔（秒）
	Registry  *registry.Registry         // 消息处理器注册器
}

// Create 创建新 Socket 并自动加入到 Sockets 管理器。
// 参数 conn: 底层网络连接。
// 返回值:
//   - socket: 创建的 Socket 实例
//   - err: 错误信息
func (ss *Sockets) Create(conn listener.Conn) (socket *Socket, err error) {
	if scc.Stopped() {
		return nil, errors.New("server closed")
	}
	socket = &Socket{sockets: ss}
	socket.id = atomic.AddUint64(&ss.index, 1)
	socket.cwrite = make(chan message.Message, Options.WriteChanSize)
	socket.connect(conn)
	ss.sockets.Store(socket.id, socket)
	ss.Emit(EventTypeConnected, socket)
	return
}

// Get 通过 Socket ID 获取 Socket。
// 参数 id: Socket 的唯一标识符。
// 返回值: 查找到的 Socket 实例，如果不存在则返回 nil。
func (ss *Sockets) Get(id uint64) (socket *Socket) {
	if i, ok := ss.sockets.Load(id); ok {
		socket, _ = i.(*Socket)
	}
	return
}

// Range 遍历所有 Socket 连接。
// 参数 fn: 遍历回调函数，如果返回 false 则停止遍历。
func (ss *Sockets) Range(fn func(socket *Socket) bool) {
	ss.sockets.Range(func(_, v any) bool {
		if s, ok := v.(*Socket); ok {
			return fn(s)
		}
		return true
	})
}

// Service 创建或获取服务。
// 参数 name: 服务名称，为空则创建默认服务。
// 返回值: 服务实例。
func (ss *Sockets) Service(name ...string) *registry.Service {
	handler := &Handler{}
	var s string
	if len(name) > 0 {
		s = name[0]
	}
	service := ss.Registry.Service(s, handler)
	service.SetMethods([]string{RegistryMethod})
	return service
}

// Register 使用默认 Service 注册接口。
// 参数:
//   - i: 要注册的处理对象
//   - prefix: 路径前缀，可选
//
// 返回值: 错误信息
func (ss *Sockets) Register(i interface{}, prefix ...string) error {
	service := ss.Service("")
	return service.Register(i, prefix...)
}

// Broadcast 广播消息到所有 Socket。
// 参数:
//   - m: 要广播的消息
//   - filter: 过滤函数，如果不为 nil 且返回 false 则跳过该 Socket
func (ss *Sockets) Broadcast(m message.Message, filter func(*Socket) bool) {
	ss.Range(func(sock *Socket) bool {
		if filter == nil || filter(sock) {
			_ = sock.Async(m)
		}
		return true
	})
}

// On 注册事件处理函数（初始化时使用）。
// 参数:
//   - e: 事件类型
//   - f: 事件处理函数
func (ss *Sockets) On(e EventType, f EventsFunc) {
	ss.emitter[e] = append(ss.emitter[e], f)
}

// Emit 触发事件。
// 参数:
//   - e: 事件类型
//   - s: 触发事件的 Socket
//   - attach: 事件附加数据，可选
func (ss *Sockets) Emit(e EventType, s *Socket, attach ...any) {
	var v any
	if len(attach) > 0 {
		v = attach[0]
	}
	for _, f := range ss.emitter[e] {
		f(s, v)
	}
}

// Errorf 抛出一个异常事件。
// 参数:
//   - s: 触发错误的 Socket
//   - format: 错误格式字符串
//   - args: 格式参数
func (ss *Sockets) Errorf(s *Socket, format any, args ...any) {
	defer func() {
		if e := recover(); e != nil {
			logger.Error(e)
		}
	}()
	var err any
	if len(args) > 0 {
		err = values.Sprintf(format, args...)
	} else {
		err = format
	}
	ss.Emit(EventTypeError, s, err)
}

// Listen 监听指定地址。
// 参数:
//   - address: 监听地址
//   - tlsConfig: TLS 配置，仅用于 WebSocket，可选
//
// 返回值:
//   - listener: 监听器实例
//   - err: 错误信息
func (ss *Sockets) Listen(address string, tlsConfig ...*tls.Config) (listener listener.Listener, err error) {
	addr := utils.NewAddress(address)
	if addr.Scheme == "" {
		addr.Scheme = "tcp"
	}
	network := strings.ToLower(addr.Scheme)
	switch network {
	case "tcp", "tcp4", "tcp6":
		listener, err = tcp.New(network, addr.String())
	case "ws", "wss", "wss4", "wss5", "wss6":
		listener, err = wss.New(network, addr.String(), tlsConfig...)
	case "udp", "udp4", "udp6":
		listener, err = udp.New(network, addr.String())
	default:
		err = errors.New("address scheme unknown")
	}
	if err != nil {
		return
	}
	ss.Accept(listener)
	return
}

// Accept 接受监听器的连接请求。
// 参数 ln: 监听器实例。
func (ss *Sockets) Accept(ln listener.Listener) {
	scc.CGO(func(ctx context.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Alert(err)
			}
		}()
		defer func() {
			_ = ln.Close()
		}()
		ss.instance = append(ss.instance, ln)
		for !scc.Stopped() {
			conn, err := ln.Accept()
			if err == nil {
				_, err = ss.Create(conn)
			}
			if errors.Is(err, net.ErrClosed) {
				return
			} else if err != nil {
				logger.Debug("listener.Accept Error:%v", err)
			}
		}
	})
}

// Start 当 cosgo 框架启动时调用。
// 返回值: 错误信息，如果启动失败则返回。
func (ss *Sockets) Start() error {
	if !ss.started.CompareAndSwap(false, true) {
		logger.Warn("sockets already started")
		return nil
	}
	cosgo.On(cosgo.EventTypClosing, ss.stop)
	scc.CGO(ss.daemon)
	return nil
}

// stop 当 cosgo 框架关闭时调用。
// 返回值: 错误信息，如果关闭失败则返回。
func (ss *Sockets) stop() error {
	for _, ln := range ss.instance {
		_ = ln.Close()
	}
	return nil
}

// Address 获取本地服务器地址。
// 返回值: 服务器地址字符串，如果未监听则返回空字符串。
func (ss *Sockets) Address() string {
	for _, l := range ss.instance {
		addr := l.Addr()
		if addr != nil {
			address := addr.String()
			// 处理特殊地址格式
			if ip, port, err := net.SplitHostPort(address); err == nil {
				switch ip {
				case "", "0.0.0.0", "127.0.0.1", "localhost":
					// 获取内网地址
					if localIP, err := utils.LocalIPv4(); err == nil && localIP != "" {
						return net.JoinHostPort(localIP, port)
					}
				}
				return address
			}
		}
	}
	return ""
}

// Connect 连接服务器。
// 参数 address: 服务器地址。
// 返回值:
//   - socket: 连接的 Socket 实例
//   - err: 错误信息
func (ss *Sockets) Connect(address string) (socket *Socket, err error) {
	conn, err := ss.tryConnect(address)
	if err != nil {
		return nil, err
	}
	socket, err = ss.Create(conn)
	if err == nil {
		socket.address = &address
	}
	return
}

// tryConnect 尝试连接到指定地址。
// 参数 s: 地址字符串。
// 返回值:
//   - conn: 连接对象
//   - err: 错误信息
func (ss *Sockets) tryConnect(s string) (listener.Conn, error) {
	address := utils.NewAddress(s)
	if address.Scheme == "" {
		address.Scheme = "tcp"
	}
	rs := address.String()
	for try := int32(0); try < Options.ClientReconnectMax; try++ {
		conn, err := net.DialTimeout(address.Scheme, rs, time.Second)
		if err == nil {
			return tcp.NewConn(conn), nil
		}
		logger.Debug("try connect server[%d],error:%v", try, err)
		time.Sleep(time.Duration(Options.ClientReconnectTime))
	}
	return nil, fmt.Errorf("failed to dial %v", rs)
}

// daemon 启动协程定时清理无效用户。
// 参数 ctx: 上下文，用于控制协程退出。
// 注意: Options.Heartbeat == 0 时不启动计时器，由业务层直接调用 Heartbeat。
func (ss *Sockets) daemon(ctx context.Context) {
	if ss.heartbeat == 0 {
		logger.Debug("sockets heartbeat not start,because heartbeat is zero")
		return
	}
	t := time.Second * time.Duration(ss.heartbeat)
	ticker := time.NewTimer(t)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ss.Heartbeat(ss.heartbeat)
			ticker.Reset(t)
		}
	}
}

// Heartbeat 对所有连接执行心跳检查。
// 参数 v: 心跳计数增量。
func (ss *Sockets) Heartbeat(v int32) {
	ss.Range(func(socket *Socket) bool {
		_ = socket.Heartbeat(v)
		return true
	})
}

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

// socket 管理,包含服务器和客户端

// Matcher 检查输入流是否包含有效的消息魔术数字
// 参数:
//
//	r: 输入流
//
// 返回值:
//
//	是否包含有效的消息魔术数字
func Matcher(r io.Reader) bool {
	buf := make([]byte, 1)
	n, _ := r.Read(buf)
	return n == 1 && message.Magics.Has(buf[0])
}

type Cosnet struct {
	index     uint64 // Socket 索引计数器
	started   atomic.Bool
	sockets   syncmap.Map                // 存储所有 Socket 连接
	emitter   map[EventType][]EventsFunc // 事件监听器映射
	instance  []listener.Listener        // 监听器实例列表
	heartbeat int32                      //心跳间隔(s)
	Registry  *registry.Registry         // 消息处理器注册器
}

func New(heartbeat int32) *Cosnet {
	ss := &Cosnet{
		index:     0,
		sockets:   syncmap.Map{},
		emitter:   make(map[EventType][]EventsFunc),
		instance:  make([]listener.Listener, 0),
		heartbeat: heartbeat,
		Registry:  registry.New(),
	}
	return ss
}

// NewSocket 创建新socket并自动加入到Sockets管理器
func (cos *Cosnet) NewSocket(conn listener.Conn) (socket *Socket, err error) {
	if scc.Stopped() {
		return nil, errors.New("server closed")
	}
	socket = &Socket{cosnet: cos}
	socket.id = atomic.AddUint64(&cos.index, 1)
	socket.cwrite = make(chan message.Message, Options.WriteChanSize)
	socket.connect(conn)
	cos.sockets.Store(socket.id, socket)
	cos.Emit(EventTypeConnected, socket)
	return
}

// Get 通过SOCKET ID获取SOCKET
// id.(string) 通过用户ID获取
// id.(MID) 通过SOCKET ID获取
func (cos *Cosnet) Get(id uint64) (socket *Socket) {
	if i, ok := cos.sockets.Load(id); ok {
		socket, _ = i.(*Socket)
	}
	return
}

// Range 遍历所有 Socket 连接
// 参数:
//
//	fn: 遍历回调函数，如果返回 false 则停止遍历
func (cos *Cosnet) Range(fn func(socket *Socket) bool) {
	cos.sockets.Range(func(_, v any) bool {
		if s, ok := v.(*Socket); ok {
			return fn(s)
		}
		return true
	})
}

// Service 创建或获取服务
// 参数:
//
//	name: 服务名称，为空则创建默认服务
//
// 返回值:
//
//	服务实例
func (cos *Cosnet) Service(name ...string) *registry.Service {
	handler := &Handler{}
	var s string
	if len(name) > 0 {
		s = name[0]
	}
	service := cos.Registry.Service(s, handler)
	service.SetMethods([]string{RegistryMethod})
	return service
}

// Register 使用默认Service注册接口
func (cos *Cosnet) Register(i interface{}, prefix ...string) error {
	service := cos.Service("")
	return service.Register(i, prefix...)
}

// Broadcast 广播,filter 过滤函数，如果不为nil且返回false则不对当期socket进行发送消息
func (cos *Cosnet) Broadcast(m message.Message, filter func(*Socket) bool) {
	cos.Range(func(sock *Socket) bool {
		if filter == nil || filter(sock) {
			_ = sock.Async(m)
		}
		return true
	})
}

// On 注册事件处理函数,初始化时使用
// 参数:
//
//	e: 事件类型
//	f: 事件处理函数
func (cos *Cosnet) On(e EventType, f EventsFunc) {
	cos.emitter[e] = append(cos.emitter[e], f)
}

// Emit 触发事件
// 参数:
//
//	e: 事件类型
//	s: 触发事件的Socket
//	attach: 事件附加数据
func (cos *Cosnet) Emit(e EventType, s *Socket, attach ...any) {
	var v any
	if len(attach) > 0 {
		v = attach[0]
	}
	for _, f := range cos.emitter[e] {
		f(s, v)
	}
}

// Errorf 抛出一个异常事件
// 参数:
//
//	s: 触发错误的Socket
//	format: 错误格式
//	args: 错误参数
func (cos *Cosnet) Errorf(s *Socket, format any, args ...any) {
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
	cos.Emit(EventTypeError, s, err)
}

// Listen 监听address
// tlsConfig 仅仅提供给websocket
func (cos *Cosnet) Listen(address string, tlsConfig ...*tls.Config) (listener listener.Listener, err error) {
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
	//case "unix", "unixgram", "unixpacket":
	default:
		err = errors.New("address scheme unknown")
	}
	if err != nil {
		return
	}
	cos.Accept(listener)
	return
}

// Accept 接受监听器的连接请求
// 参数:
//
//	ln: 监听器
func (cos *Cosnet) Accept(ln listener.Listener) {
	scc.CGO(func(ctx context.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.Alert(err)
			}
		}()
		defer func() {
			_ = ln.Close()
		}()
		cos.instance = append(cos.instance, ln)
		for !scc.Stopped() {
			conn, err := ln.Accept()
			if err == nil {
				_, err = cos.NewSocket(conn)
			}
			if errors.Is(err, net.ErrClosed) {
				return
			} else if err != nil {
				logger.Debug("listener.Accept Error:%v", err)
			}
		}
	})
}

// Start 当 cosgo 框架启动时调用
// 返回值:
//
//	错误信息，如果启动失败则返回
func (cos *Cosnet) Start() error {
	if !cos.started.CompareAndSwap(false, true) {
		logger.Warn("cosnet already started")
		return nil
	}
	cosgo.On(cosgo.EventTypClosing, cos.stop)
	scc.CGO(cos.daemon)
	return nil
}

// stop 当 cosgo 框架关闭时调用
// 返回值:
//
//	错误信息，如果关闭失败则返回
func (cos *Cosnet) stop() error {
	for _, ln := range cos.instance {
		_ = ln.Close()
	}
	return nil
}

// Connect 连接服务器address
func (cos *Cosnet) Connect(address string) (socket *Socket, err error) {
	conn, err := cos.tryConnect(address)
	if err != nil {
		return nil, err
	}
	socket, err = cos.NewSocket(conn)
	if err == nil {
		socket.address = &address
	}
	return
}

// tryConnect 尝试连接到指定地址
// 参数:
//
//	s: 地址字符串
//
// 返回值:
//
//	连接对象和错误信息
func (cos *Cosnet) tryConnect(s string) (listener.Conn, error) {
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

// daemon 启动协程定时清理无效用户
// 参数:
//
//	ctx: 上下文，用于控制协程退出
//
// 注意:
//
//	Options.Heartbeat == 0 时不启动计时器，由业务层直接调用Heartbeat
func (cos *Cosnet) daemon(ctx context.Context) {
	if cos.heartbeat == 0 {
		logger.Debug("cosnet heartbeat not start,because heartbeat is zero")
		return
	}
	// 计算心跳间隔
	t := time.Second * time.Duration(cos.heartbeat)
	ticker := time.NewTimer(t)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 执行心跳检查
			cos.Heartbeat(cos.heartbeat)
			// 重置计时器
			ticker.Reset(t)
		}
	}
}

// Heartbeat 对所有连接执行心跳检查
// 参数:
//
//	v: 心跳计数增量
func (cos *Cosnet) Heartbeat(v int32) {
	cos.Range(func(socket *Socket) bool {
		_ = socket.Heartbeat(v)
		return true
	})
}

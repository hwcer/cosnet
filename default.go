package cosnet

import (
	"crypto/tls"

	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/cosnet/message"
)

var Default = New(Options.Heartbeat)

// NewSocket 创建新socket并自动加入到Sockets管理器
func NewSocket(conn listener.Conn) (socket *Socket, err error) {
	return Default.NewSocket(conn)
}

// onStart 当 cosgo 框架启动时调用
// 返回值:
//
//	错误信息，如果启动失败则返回

func Start() error {
	Default.heartbeat = Options.Heartbeat
	return Default.Start()
}

func On(eventType EventType, eventFunc func(*Socket, any)) {
	Default.On(eventType, eventFunc)
}

// Get 通过SOCKET ID获取SOCKET
// id.(string) 通过用户ID获取
// id.(MID) 通过SOCKET ID获取
func Get(id uint64) (socket *Socket) {
	if i, ok := Default.sockets.Load(id); ok {
		socket, _ = i.(*Socket)
	}
	return
}

// Range 遍历所有 Socket 连接
// 参数:
//
//	fn: 遍历回调函数，如果返回 false 则停止遍历
func Range(fn func(socket *Socket) bool) {
	Default.sockets.Range(func(_, v any) bool {
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
func Service(name ...string) *registry.Service {
	handler := &Handler{}
	var s string
	if len(name) > 0 {
		s = name[0]
	}
	service := Default.Registry.Service(s, handler)
	service.SetMethods([]string{RegistryMethod})
	return service
}

// Register 使用默认Service注册接口
func Register(i interface{}, prefix ...string) error {
	service := Service("")
	return service.Register(i, prefix...)
}

// Broadcast 广播,filter 过滤函数，如果不为nil且返回false则不对当期socket进行发送消息
func Broadcast(m message.Message, filter func(*Socket) bool) {
	Default.Range(func(sock *Socket) bool {
		if filter == nil || filter(sock) {
			_ = sock.Async(m)
		}
		return true
	})
}

// Listen 监听address
// tlsConfig 仅仅提供给websocket
func Listen(address string, tlsConfig ...*tls.Config) (listener listener.Listener, err error) {
	return Default.Listen(address, tlsConfig...)
}

// Accept 接受监听器的连接请求
// 参数:
//
//	ln: 监听器
func Accept(ln listener.Listener) {
	Default.Accept(ln)
}

// Connect 连接服务器address
func Connect(address string) (socket *Socket, err error) {
	return Default.Connect(address)
}

// Heartbeat 对所有连接执行心跳检查
// 参数:
//
//	v: 心跳计数增量
func Heartbeat(v int32) {
	Default.Heartbeat(v)
}

package cosnet

import (
	"crypto/tls"

	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/cosnet/message"
)

// Default 是默认的 Sockets 实例。
var Default = New()

// Create 创建新 Socket 并自动加入到默认 NetHub 管理器。
// 参数 conn: 底层网络连接。
// 返回值:
//   - socket: 创建的 Socket 实例
//   - err: 错误信息
func Create(conn listener.Conn) (socket *Socket, err error) {
	return Default.Create(conn)
}

// Start 当 cosgo 框架启动时调用。
// 返回值: 错误信息，如果启动失败则返回。
func Start() error {
	return Default.Start()
}

// On 注册事件处理函数到默认实例。
// 参数:
//   - eventType: 事件类型
//   - eventFunc: 事件处理函数
func On(eventType EventType, eventFunc func(*Socket, any)) {
	Default.On(eventType, eventFunc)
}

// Get 通过 Socket ID 从默认实例获取 Socket。
// 参数 id: Socket 的唯一标识符。
// 返回值: 查找到的 Socket 实例，如果不存在则返回 nil。
func Get(id uint64) (socket *Socket) {
	if i, ok := Default.sockets.Load(id); ok {
		socket, _ = i.(*Socket)
	}
	return
}

// Range 遍历默认实例中的所有 Socket 连接。
// 参数 fn: 遍历回调函数，如果返回 false 则停止遍历。
func Range(fn func(socket *Socket) bool) {
	Default.sockets.Range(func(_, v any) bool {
		if s, ok := v.(*Socket); ok {
			return fn(s)
		}
		return true
	})
}

// Service 创建或获取服务（默认实例）。
// 参数 name: 服务名称，为空则创建默认服务。
// 返回值: 服务实例。
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

// Register 使用默认 Service 注册接口。
// 参数:
//   - i: 要注册的处理对象
//   - prefix: 路径前缀，可选
//
// 返回值: 错误信息
func Register(i interface{}, prefix ...string) error {
	service := Service("")
	return service.Register(i, prefix...)
}

// Broadcast 广播消息到默认实例中的所有 Socket。
// 参数:
//   - m: 要广播的消息
//   - filter: 过滤函数，如果不为 nil 且返回 false 则跳过该 Socket
func Broadcast(m message.Message, filter func(*Socket) bool) {
	Default.Range(func(sock *Socket) bool {
		if filter == nil || filter(sock) {
			_ = sock.Async(m)
		}
		return true
	})
}

// Listen 监听指定地址（默认实例）。
// 参数:
//   - address: 监听地址
//   - tlsConfig: TLS 配置，仅用于 WebSocket，可选
//
// 返回值:
//   - listener: 监听器实例
//   - err: 错误信息
func Listen(address string, tlsConfig ...*tls.Config) (listener listener.Listener, err error) {
	return Default.Listen(address, tlsConfig...)
}

// Accept 接受监听器的连接请求（默认实例）。
// 参数 ln: 监听器实例。
func Accept(ln listener.Listener) {
	Default.Accept(ln)
}

// Connect 连接服务器（默认实例）。
// 参数 address: 服务器地址。
// 返回值:
//   - socket: 连接的 Socket 实例
//   - err: 错误信息
func Connect(address string) (socket *Socket, err error) {
	return Default.Connect(address)
}

// Heartbeat 对默认实例中的所有连接执行心跳检查。
// 参数 v: 心跳计数增量。
func Heartbeat(v int32) {
	Default.Heartbeat(v)
}

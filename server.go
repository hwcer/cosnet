package cosnet

import (
	"errors"

	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/cosnet/message"
	"golang.org/x/sync/syncmap"
)

var (
	index    uint64                     // Socket 索引计数器
	online   int32                      // 当前在线人数
	sockets  syncmap.Map                // 存储所有 Socket 连接
	emitter  map[EventType][]EventsFunc // 事件监听器映射
	instance []listener.Listener        // 监听器实例列表
	Registry *registry.Registry         // 消息处理器注册器
)

func init() {
	sockets = syncmap.Map{}
	emitter = make(map[EventType][]EventsFunc)
	Registry = registry.New()
}

// New 创建新socket并自动加入到Sockets管理器
func New(conn listener.Conn) (socket *Socket, err error) {
	if scc.Stopped() {
		return nil, errors.New("server closed")
	}
	socket = NewSocket(conn)
	sockets.Store(socket.id, socket)
	Emit(EventTypeConnected, socket)
	return
}

// Get 通过SOCKET ID获取SOCKET
// id.(string) 通过用户ID获取
// id.(MID) 通过SOCKET ID获取
func Get(id uint64) (socket *Socket) {
	if i, ok := sockets.Load(id); ok {
		socket, _ = i.(*Socket)
	}
	return
}

// Online 当前在线总人数
func Online() int32 {
	return online
}

// Range 遍历所有 Socket 连接
// 参数:
//
//	fn: 遍历回调函数，如果返回 false 则停止遍历
func Range(fn func(socket *Socket) bool) {
	sockets.Range(func(_, v any) bool {
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
	service := Registry.Service(s, handler)
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
	Range(func(sock *Socket) bool {
		if filter == nil || filter(sock) {
			_ = sock.Async(m)
		}
		return true
	})
}

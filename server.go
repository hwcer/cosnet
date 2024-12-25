package cosnet

import (
	"errors"
	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosnet/listener"
	"golang.org/x/sync/syncmap"
)

var (
	index    uint64
	online   int32
	sockets  syncmap.Map                //存储Socket
	emitter  map[EventType][]EventsFunc //事件监听
	instance []listener.Listener
	Registry *registry.Registry //注册器
)

func init() {
	sockets = syncmap.Map{}
	emitter = make(map[EventType][]EventsFunc)
	Registry = registry.New(nil)
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

func Range(fn func(socket *Socket) bool) {
	sockets.Range(func(_, v any) bool {
		if s, ok := v.(*Socket); ok {
			return fn(s)
		}
		return true
	})
}

func Service(name string, handler ...interface{}) *registry.Service {
	service := Registry.Service(name)
	if service.Handler == nil {
		service.Handler = &Handler{}
	}
	if h, ok := service.Handler.(*Handler); ok {
		for _, i := range handler {
			h.Use(i)
		}
	}
	return service
}

// Register 使用默认Service注册接口
func Register(i interface{}, prefix ...string) error {
	service := Service("")
	return service.Register(i, prefix...)
}

// Broadcast 广播,filter 过滤函数，如果不为nil且返回false则不对当期socket进行发送消息
func Broadcast(path string, data any, filter func(*Socket) bool) {
	Range(func(sock *Socket) bool {
		if filter == nil || filter(sock) {
			_ = sock.Send(path, nil, data)
		}
		return true
	})
}

package cosnet

import (
	"context"
	"fmt"
	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosgo/storage"
	"github.com/hwcer/cosgo/utils"
	"net"
	"runtime/debug"
	"strings"
	"time"
)

func newSetter(id storage.MID, val interface{}) storage.Setter {
	d := val.(*Socket)
	d.Data = *storage.NewData(id, nil)
	return d
}

func New(ctx context.Context) *Server {
	i := &Server{
		scc:      utils.NewSCC(ctx),
		listener: make(map[EventType][]EventsFunc),
		registry: registry.New(nil),
	}
	i.Players = NewPlayers()
	i.Sockets = storage.New(1024)
	i.Sockets.NewSetter = newSetter
	i.scc.CGO(i.heartbeat)
	return i
}

type Server struct {
	scc      *utils.SCC
	Players  *Players //存储用户登录信息
	Sockets  *storage.Array
	listener map[EventType][]EventsFunc //事件监听
	registry *registry.Registry
}

func (this *Server) GO(f func()) {
	this.scc.GO(f)
}

func (this *Server) CGO(f func(ctx context.Context)) {
	this.scc.CGO(f)
}

func (this *Server) SCC() *utils.SCC {
	return this.scc
}

func (this *Server) Stopped() bool {
	return this.scc.Stopped()
}

func (this *Server) Size() int {
	return this.Sockets.Size()
}

// New 创建新socket并自动加入到Sockets管理器
func (this *Server) New(conn net.Conn, netType NetType) (socket *Socket, err error) {
	socket = NewSocket(this, conn, netType)
	this.Sockets.Create(socket)
	this.Emit(EventTypeConnected, socket)
	return
}

// Remove 彻底销毁,移除资源
func (this *Server) Remove(socket *Socket) {
	defer func() { _ = recover() }()
	socket.status.Destroy(func(r bool) {
		if !r {
			return
		}
		this.Players.Remove(socket)
		this.Sockets.Remove(socket.Id())
		socket.emit(EventTypeDestroyed)
	})
}

func (this *Server) Range(fn func(socket *Socket) bool) {
	this.Sockets.Range(func(v storage.Setter) bool {
		if s, ok := v.(*Socket); ok {
			return fn(s)
		}
		return true
	})
}

func (this *Server) Service(name string, handler ...interface{}) *registry.Service {
	service := this.registry.Service(name)
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

func (this *Server) Register(i interface{}, prefix ...string) error {
	service := this.Service("")
	return service.Register(i, prefix...)
}

func (this *Server) Close(timeout time.Duration) error {
	if !this.scc.Cancel() {
		return nil
	}
	return this.scc.Wait(timeout)
}

// Socket 通过SOCKETID获取SOCKET
// id.(string) 通过用户ID获取
// id.(MID) 通过SOCKET ID获取
func (this *Server) Socket(id any) (socket *Socket) {
	switch v := id.(type) {
	case string:
		if player := this.Players.Get(v); player != nil {
			socket = player.socket
		}
	case storage.MID:
		if i, ok := this.Sockets.Get(v); ok {
			socket, _ = i.(*Socket)
		}
	}
	return
}

// Player 获取用户对象,id同this.Socket(id)
func (this *Server) Player(id any) (player *Player) {
	switch v := id.(type) {
	case string:
		player = this.Players.Get(v)
	case storage.MID:
		if socket, ok := this.Sockets.Get(v); ok {
			if r := socket.Get(); r != nil {
				player, _ = r.(*Player)
			}
		}
	}
	return
}

// Broadcast 广播,filter 过滤函数，如果不为nil且返回false则不对当期socket进行发送消息
func (this *Server) Broadcast(path string, data any, filter func(*Socket) bool) {
	this.Range(func(sock *Socket) bool {
		if filter == nil || filter(sock) {
			_ = sock.Send(0, path, data)
		}
		return true
	})
}

func (this *Server) handle(socket *Socket, msg *Message) {
	var err error
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("server handle error:%v\n%v", e, string(debug.Stack()))
		}
		if err != nil {
			this.Errorf(socket, err)
		}
	}()

	path := msg.Path()
	if i := strings.Index(path, "?"); i >= 0 {
		path = path[0:i]
	}
	node, ok := this.registry.Match(path)
	if !ok {
		this.Emit(EventTypeMessage, socket, msg)
		return
	}
	handler := node.Service.Handler.(*Handler)
	if handler == nil {
		return
	}
	c := &Context{Socket: socket, Message: msg}
	var reply interface{}
	reply, err = handler.Caller(node, c)
	if err != nil {
		return
	}
	err = handler.Serialize(c, reply)
}

// 11v9
// heartbeat 启动协程定时清理无效用户
func (this *Server) heartbeat(ctx context.Context) {
	t := time.Millisecond * time.Duration(Options.SocketHeartbeat)
	ticker := time.NewTimer(t)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			utils.Try(this.doHeartbeat)
			ticker.Reset(t)
		}
	}
}
func (this *Server) doHeartbeat() {
	this.Range(func(socket *Socket) bool {
		socket.Heartbeat()
		this.Emit(EventTypeHeartbeat, socket)
		return true
	})
}

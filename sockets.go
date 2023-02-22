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

func New(ctx context.Context) *Sockets {
	i := &Sockets{
		scc: utils.NewSCC(ctx),
		//pool:     sync.Pool{},
		Array:    *storage.New(1024),
		listener: make(map[EventType][]EventsFunc),
		registry: registry.New(nil),
	}
	//i.pool.New = func() interface{} {
	//	return &Message{}
	//}
	//i.Binder = binder.New(binder.MIMEJSON)
	//i.Players = NewPlayers(i)
	i.Array.NewSetter = newSetter
	i.scc.CGO(i.heartbeat)
	return i
}

type Sockets struct {
	storage.Array
	scc *utils.SCC
	//pool     sync.Pool
	//Players  *Players                   //存储用户登录信息
	listener map[EventType][]EventsFunc //事件监听
	registry *registry.Registry
}

func (this *Sockets) GO(f func()) {
	this.scc.GO(f)
}

func (this *Sockets) CGO(f func(ctx context.Context)) {
	this.scc.CGO(f)
}

func (this *Sockets) SCC() *utils.SCC {
	return this.scc
}

func (this *Sockets) Stopped() bool {
	return this.scc.Stopped()
}

func (this *Sockets) Size() int {
	return this.Array.Size()
}

// New 创建新socket并自动加入到Sockets管理器
func (this *Sockets) New(conn net.Conn, netType NetType) (socket *Socket, err error) {
	socket = NewSocket(this, conn, netType)
	this.Array.Create(socket)
	this.Emit(EventTypeConnected, socket)
	return
}

func (this *Sockets) Range(fn func(socket *Socket) bool) {
	this.Array.Range(func(v storage.Setter) bool {
		if s, ok := v.(*Socket); ok {
			return fn(s)
		}
		return true
	})
}

func (this *Sockets) Service(name string, handler ...interface{}) *registry.Service {
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

func (this *Sockets) Register(i interface{}, prefix ...string) error {
	service := this.Service("")
	return service.Register(i, prefix...)
}

func (this *Sockets) Close(timeout time.Duration) error {
	if !this.scc.Cancel() {
		return nil
	}
	return this.scc.Wait(timeout)
}

// Socket 通过SOCKETID获取SOCKET
// id.(string) 通过用户ID获取
// id.(MID) 通过SOCKET ID获取
func (this *Sockets) Socket(id storage.MID) (socket *Socket) {
	if r, ok := this.Array.Get(id); ok {
		socket, _ = r.(*Socket)
	}
	return
}

// Player 获取用户对象,id同this.Socket(id)
//func (this *Sockets) Player(id interface{}) (player *Player) {
//	switch id.(type) {
//	case string:
//		player = this.Players.Player(id.(string))
//	case storage.MID:
//		if d, ok := this.Array.Get(id.(storage.MID)); ok {
//			r := d.Get()
//			player, _ = r.(*Player)
//		}
//	}
//	return
//}

// Broadcast 广播,filter 过滤函数，如果不为nil且返回false则不对当期socket进行发送消息
func (this *Sockets) Broadcast(path string, data any, filter func(*Socket) bool) {
	this.Range(func(sock *Socket) bool {
		if filter == nil || filter(sock) {
			_ = sock.Send(0, path, data)
		}
		return true
	})
}

func (this *Sockets) handle(socket *Socket, msg *Message) {
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
func (this *Sockets) heartbeat(ctx context.Context) {
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
func (this *Sockets) doHeartbeat() {
	this.Range(func(socket *Socket) bool {
		socket.Heartbeat()
		this.Emit(EventTypeHeartbeat, socket)
		return true
	})
}

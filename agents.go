package cosnet

import (
	"context"
	"errors"
	"github.com/hwcer/cosgo/smap"
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/logger"
	"github.com/hwcer/registry"
	"net"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

func newSetter(id smap.MID, val interface{}) smap.Setter {
	d := val.(*Socket)
	d.Data = smap.NewData(id, nil)
	return d
}

func New(ctx context.Context) *Agents {
	i := &Agents{
		scc:      utils.NewSCC(ctx),
		pool:     sync.Pool{},
		Array:    smap.New(1024),
		listener: make(map[EventType][]EventsFunc),
		registry: registry.New(nil),
	}
	i.pool.New = func() interface{} {
		return &Message{}
	}
	i.Players = NewPlayers(i)
	i.Array.NewSetter = newSetter
	i.scc.CGO(i.heartbeat)
	return i
}

type Agents struct {
	*smap.Array
	scc      *utils.SCC
	pool     sync.Pool
	Handle   func(*Context) (interface{}, error) //默认消息处理器，registry匹配不到时使用
	Players  *Players                            //存储用户登录信息
	listener map[EventType][]EventsFunc          //事件监听
	registry *registry.Registry
}

func (this *Agents) GO(f func()) {
	this.scc.GO(f)
}

func (this *Agents) CGO(f func(ctx context.Context)) {
	this.scc.CGO(f)
}

func (this *Agents) SCC() *utils.SCC {
	return this.scc
}

func (this *Agents) Stopped() bool {
	return this.scc.Stopped()
}

func (this *Agents) Size() int {
	return this.Array.Size()
}

// New 创建新socket并自动加入到Sockets管理器
func (this *Agents) New(conn net.Conn, netType NetType) (socket *Socket, err error) {
	socket = NewSocket(this, conn, netType)
	this.Array.Create(socket)
	this.Emit(EventTypeConnected, socket)
	return
}

func (this *Agents) Range(fn func(socket *Socket) bool) {
	this.Array.Range(func(v smap.Setter) bool {
		if s, ok := v.(*Socket); ok {
			return fn(s)
		}
		return true
	})
}

func (this *Agents) Acquire() *Message {
	r, _ := this.pool.Get().(*Message)
	return r
}

func (this *Agents) Release(i *Message) {
	i.code = 0
	i.path = 0
	i.body = 0
	this.pool.Put(i)
}

func (this *Agents) Service(name string, handler ...interface{}) *registry.Service {
	service := this.registry.Service(name)
	if service.Handler == nil {
		h := &Handler{}
		service.Handler = h
		service.On(registry.FilterEventTypeFunc, h.Filter)
		service.On(registry.FilterEventTypeMethod, h.Filter)
		service.On(registry.FilterEventTypeStruct, h.Filter)
	}
	if h, ok := service.Handler.(*Handler); ok {
		for _, i := range handler {
			h.Use(i)
		}
	}
	return service
}
func (this *Agents) Register(f interface{}, prefix ...string) error {
	service := this.Service("")
	return service.Register(f, prefix...)
}

func (this *Agents) Close(timeout time.Duration) error {
	if !this.scc.Cancel() {
		return nil
	}
	return this.scc.Wait(timeout)
}

// Socket 通过SOCKETID获取SOCKET
// id.(string) 通过用户ID获取
// id.(MID) 通过SOCKET ID获取
func (this *Agents) Socket(id interface{}) (socket *Socket) {
	switch id.(type) {
	case string:
		socket = this.Players.Socket(id.(string))
	case smap.MID:
		if r, ok := this.Array.Get(id.(smap.MID)); ok {
			socket, _ = r.(*Socket)
		}
	}
	return
}

// Player 获取用户对象,id同this.Socket(id)
func (this *Agents) Player(id interface{}) (player *Player) {
	switch id.(type) {
	case string:
		player = this.Players.Player(id.(string))
	case smap.MID:
		if d, ok := this.Array.Get(id.(smap.MID)); ok {
			r := d.Get()
			player, _ = r.(*Player)
		}
	}
	return
}

// Broadcast 广播,filter 过滤函数，如果不为nil且返回false则不对当期socket进行发送消息
func (this *Agents) Broadcast(msg *Message, filter func(*Socket) bool) {
	this.Range(func(sock *Socket) bool {
		if filter == nil || filter(sock) {
			sock.Write(msg)
		}
		return true
	})
}

func (this *Agents) handle(socket *Socket, msg *Message) (err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Info("server recover error:%v\n%v", r, string(debug.Stack()))
		}
	}()
	c := &Context{Socket: socket, Message: msg}
	path := c.Path()
	if i := strings.Index(path, "?"); i >= 0 {
		path = path[0:i]
	}
	urlPath := this.registry.Clean(path)
	node, ok := this.registry.Match(urlPath)
	var reply interface{}
	if !ok {
		if this.Handle != nil {
			reply, err = this.Handle(c)
		} else {
			err = errors.New("ServicePath not exist")
		}
		if err != nil {
			return err
		}
		return Serialize(c, reply)
	} else {
		service := node.Service()
		handler, _ := service.Handler.(*Handler)
		if !ok {
			return errors.New("handler unknown")
		}
		reply, err = handler.Caller(node, c)
		if err != nil {
			return err
		}
		return handler.Serialize(c, reply)
	}
}

// heartbeat 启动协程定时清理无效用户
func (this *Agents) heartbeat(ctx context.Context) {
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
func (this *Agents) doHeartbeat() {
	this.Range(func(socket *Socket) bool {
		socket.Heartbeat()
		this.Emit(EventTypeHeartbeat, socket)
		return true
	})
}

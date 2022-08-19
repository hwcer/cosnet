package sockets

import (
	"context"
	"errors"
	"github.com/hwcer/cosgo/smap"
	"github.com/hwcer/cosgo/utils"
	"net"
	"time"
)

func newSetter(id smap.MID, val interface{}) smap.Setter {
	d := val.(*Socket)
	d.Data = smap.NewData(id, nil)
	return d
}

func New(ctx context.Context, handler Handler) *Agents {
	i := &Agents{
		scc:      utils.NewSCC(ctx),
		Array:    smap.New(1024),
		Handler:  handler,
		listener: make(map[EventType][]EventsFunc),
	}
	i.Players = NewPlayers(i)
	i.Array.NewSetter = newSetter
	i.scc.CGO(i.heartbeat)
	return i
}

type Agents struct {
	*smap.Array
	scc      *utils.SCC
	Players  *Players                   //存储用户登录信息
	Handler  Handler                    //默认消息处理,handle中未明确注册的消息一律进入到这里
	listener map[EventType][]EventsFunc //事件监听
}

func (this *Agents) SCC() *utils.SCC {
	return this.scc
}

func (this *Agents) Size() int {
	return this.Array.Size()
}

// New 创建新socket并自动加入到Sockets管理器
func (this *Agents) New(conn net.Conn, netType NetType) (socket *Socket, err error) {
	if this.Handler == nil {
		return nil, errors.New("handler is nil")
	}
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

func (this *Agents) Register(i interface{}) error {
	return this.Handler.Register(i)
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
func (this *Agents) Broadcast(msg Message, filter func(*Socket) bool) {
	this.Range(func(sock *Socket) bool {
		if filter == nil || filter(sock) {
			sock.Write(msg)
		}
		return true
	})
}

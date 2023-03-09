package cosnet

import (
	"sync"
)

// NewPlayers 登录之后生成 player数据
func NewPlayers() *Players {
	p := &Players{dict: sync.Map{}}
	return p
}

func NewPlayer(uuid string, socket *Socket) *Player {
	return &Player{uuid: uuid, socket: socket}
}

type Player struct {
	//values.Values
	uuid   string
	mutex  sync.Mutex
	socket *Socket
	attach any //用户登录信息,推荐存入一个struct
}

type Players struct {
	dict sync.Map
}

// replace 顶号
func (this *Player) replace(socket *Socket) {
	var old *Socket
	old, this.socket = this.socket, socket
	if old.status.Has(StatusTypeConnect) {
		old.emit(EventTypeReplaced)
		old.Close()
	}
	return
}

func (this *Player) UUID() string {
	return this.uuid
}

func (this *Player) Attach() any {
	return this.attach
}

func (this *Player) Socket() *Socket {
	return this.socket
}

// todo 并发
func (this *Players) Remove(socket *Socket) bool {
	player := socket.Player()
	if player != nil {
		this.dict.Delete(player.uuid)
	}
	return true
}

func (this *Players) Get(uuid string) *Player {
	v, ok := this.dict.Load(uuid)
	if !ok {
		return nil
	}
	p, _ := v.(*Player)
	return p
}

func (this *Players) Range(fn func(*Player) bool) {
	this.dict.Range(func(k, v interface{}) bool {
		if p, ok := v.(*Player); ok {
			return fn(p)
		}
		return true
	})
}

// Verify 身份认证,登录,TOKEN信息验证之后调用
func (this *Players) Verify(uuid string, socket *Socket, attach any) (r *Player, err error) {
	player := NewPlayer(uuid, socket)
	player.mutex.Lock()
	defer player.mutex.Unlock()
	if v, loaded := this.dict.LoadOrStore(uuid, player); loaded {
		r, _ = v.(*Player)
		if err = this.reconnect(r, socket); err != nil {
			return
		}
	} else {
		r = player
	}
	r.attach = attach
	socket.Set(r)
	socket.emit(EventTypeVerified)
	return
}

func (this *Players) reconnect(player *Player, socket *Socket) error {
	player.mutex.Lock()
	defer player.mutex.Unlock()
	//if player.socket != nil && player.socket.status.Has(StatusTypeConnect) && player.socket.cwrite != nil {
	//	player.socket.cwrite, this.cwrite = this.cwrite, player.socket.cwrite
	//}
	player.replace(socket)
	socket.emit(EventTypeReconnected)
	return nil
}

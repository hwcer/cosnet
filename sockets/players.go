package sockets

import (
	"sync"
)

// NewPlayers 登录之后生成 player数据
func NewPlayers(engine *Engine) *Players {
	p := &Players{Map: sync.Map{}}
	engine.On(EventTypeDestroyed, p.destroy)
	return p
}

type Player struct {
	uuid   string
	data   interface{} //用户数据
	secret string      //短信重连秘钥
	mutex  sync.Mutex
	socket *Socket
}

type Players struct {
	sync.Map
}

// replace 顶号
func (this *Player) replace(socket *Socket) {
	old := this.socket
	this.socket = socket
	if old.status == StatusTypeConnect {
		old.emit(EventTypeReplaced)
		old.Close()
	}
	return
}

func (this *Player) Socket() *Socket {
	return this.socket
}

func (this *Players) destroy(socket *Socket, _ interface{}) bool {
	player := socket.Player()
	if player != nil {
		this.Delete(player.uuid)
	}
	return true
}

func (this *Players) Player(uuid string) *Player {
	v, ok := this.Map.Load(uuid)
	if !ok {
		return nil
	}
	p, _ := v.(*Player)
	return p
}

func (this *Players) Socket(uuid string) *Socket {
	p := this.Player(uuid)
	if p != nil {
		return p.socket
	}
	return nil
}

func (this *Players) Range(fn func(*Player) bool) {
	this.Map.Range(func(k, v interface{}) bool {
		if p, ok := v.(*Player); ok {
			return fn(p)
		}
		return true
	})
}

// Verify 身份认证,登录,TOKEN信息验证之后调用
func (this *Socket) Verify(uuid string, data interface{}) (err error) {
	if this.Authenticated() {
		return ErrAuthDataExist
	}
	if Options.SocketReconnectTime == 0 {
		this.Set(data)
		this.emit(EventTypeVerified)
		return
	}
	player := &Player{uuid: uuid, data: data, socket: this}
	player.mutex.Lock()
	defer player.mutex.Unlock()
	if v, loaded := this.engine.Players.Map.LoadOrStore(uuid, player); loaded {
		p, _ := v.(*Player)
		err = this.reconnect(p, data)
	} else {
		this.Set(player)
		this.emit(EventTypeVerified)
	}
	return
}

func (this *Socket) reconnect(player *Player, data interface{}) error {
	if player == nil {
		return ErrAuthDataIllegal
	}
	player.mutex.Lock()
	defer player.mutex.Unlock()
	if this.Authenticated() {
		return ErrAuthDataExist
	}
	player.data = data
	if player.socket != nil && player.socket.status != StatusTypeDestroying && player.socket.cwrite != nil {
		player.socket.cwrite, this.cwrite = this.cwrite, player.socket.cwrite
	}
	player.replace(this)
	this.Set(player)
	this.emit(EventTypeReconnected)
	return nil
}

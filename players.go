package cosnet

import (
	"errors"
	"strconv"
	"strings"
	"sync"
)

func newPlayers() *players {
	return &players{}
}

type player struct {
	Id     string
	Data   interface{} //用户数据
	Socket *Socket
	mutex  sync.Mutex
}

type players struct {
	dict sync.Map
}

//replace 顶号
func (this *player) replace(socket *Socket) {
	old := this.Socket
	this.Socket = socket
	if old.status == StatusTypeNone {
		old.emit(EventTypeReplaced)
		old.Close()
	}
	return
}
func (this *players) Len() (r int) {
	this.dict.Range(func(key, value interface{}) bool {
		r++
		return true
	})
	return
}

func (this *players) Player(id string) (p *player, ok bool) {
	var v interface{}
	if v, ok = this.dict.Load(id); !ok {
		return
	}
	p, ok = v.(*player)
	return
}

func (this *players) Socket(id string) (s *Socket, ok bool) {
	var p *player
	if p, ok = this.Player(id); !ok {
		return
	}
	s = p.Socket
	return
}

func (this *players) Range(callback func(*player) bool) {
	this.dict.Range(func(k, v interface{}) bool {
		if p, ok := v.(*player); ok {
			return callback(p)
		}
		return true
	})
}

func (this *players) remove(socket *Socket) {
	if Options.SocketReconnectTime == 0 || !socket.Authenticated() {
		return
	}
	v := socket.Get()
	if v == nil {
		return
	}
	p, ok := v.(*player)
	if !ok {
		return
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.Socket.Id() == socket.Id() {
		this.dict.Delete(p.Id)
	}
}

var (
	ErrAuthDataExist   = errors.New("authenticated")
	ErrAuthDataIllegal = errors.New("authentication data illegal")
)

//Authentication 身份认证
// secret 断线重连需要使用的秘钥,需要使用协议发送给客户端
func (this *Socket) Authentication(id string, data interface{}) (secret string, err error) {
	if this.Authenticated() {
		return "", ErrAuthDataExist
	}
	if Options.SocketReconnectTime == 0 {
		this.Set(data)
		return
	}

	p := &player{Id: id, Data: data, Socket: this}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if v, loaded := this.cosnet.Players.dict.LoadOrStore(id, p); loaded {
		var ok bool
		if p, ok = v.(*player); !ok {
			return "", ErrAuthDataIllegal
		}
		p.mutex.Lock()
		defer p.mutex.Unlock()
		p.replace(this)
	}
	this.Set(p)

	k := strconv.FormatUint(this.Id(), 32)
	secret = strings.Join([]string{k, id}, "x")
	this.emit(EventTypeAuthentication)
	return
}

var (
	ErrReconnectSecretIllegal = errors.New("reconnect secret illegal")
	ErrReconnectPlayerEmpty   = errors.New("reconnect player not exist")
	ErrReconnectSecretError   = errors.New("reconnect secret error")
	ErrReconnectDisabled      = errors.New("reconnect disabled")
)

//Reconnect 断线重连,secret为Authentication时产生的
func (this *Socket) Reconnect(secret string) error {
	if this.Authenticated() {
		return ErrAuthDataExist
	}
	if Options.SocketReconnectTime == 0 {
		return ErrReconnectDisabled
	}
	index := strings.Index(secret, "x")
	if index < 0 || index >= len(secret) {
		return ErrReconnectSecretIllegal
	}
	id := secret[index+1:]
	p, ok := this.cosnet.Players.Player(id)
	if !ok {
		return ErrReconnectPlayerEmpty
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if secret[0:index] != strconv.FormatUint(p.Socket.Id(), 32) {
		return ErrReconnectSecretError
	}
	p.Socket.cwrite, this.cwrite = this.cwrite, p.Socket.cwrite
	p.replace(this)
	this.Set(p)
	this.emit(EventTypeReconnected)
	return nil
}

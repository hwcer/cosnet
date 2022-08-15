package cosnet

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrAuthDataExist   = errors.New("authenticated")
	ErrAuthDataIllegal = errors.New("authentication data illegal")

	ErrReconnectSecretIllegal = errors.New("reconnect secret illegal")
	ErrReconnectPlayerEmpty   = errors.New("reconnect Player not exist")
	ErrReconnectSecretError   = errors.New("reconnect secret error")
	ErrReconnectDisabled      = errors.New("reconnect disabled")
)

// Players 登录之后生成 player数据
func NewPlayers() *Players {
	return &Players{}
}

type Player struct {
	uuid   string
	data   interface{} //用户数据
	secret string      //短信重连秘钥
	mutex  sync.Mutex
	socket *Socket
}

type Players struct {
	dict   sync.Map
	length int32
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

// Len 在线用户
func (this *Players) Len() (r int32) {
	return this.length
}

func (this *Players) Player(uuid string) *Player {
	v, ok := this.dict.Load(uuid)
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

func (this *Players) Range(callback func(*Player) bool) {
	this.dict.Range(func(k, v interface{}) bool {
		if p, ok := v.(*Player); ok {
			return callback(p)
		}
		return true
	})
}

func (this *Players) Remove(player *Player) {
	if player == nil {
		return
	}
	this.dict.Delete(player.uuid)
}

func (this *Players) disconnect(socket *Socket) {
	atomic.AddInt32(&this.length, -1)
}

func (this *Players) reconnect(socket *Socket, uuid, secret string) error {
	player := this.Player(uuid)
	if player == nil {
		return ErrReconnectPlayerEmpty
	}
	player.mutex.Lock()
	defer player.mutex.Unlock()
	if secret != player.secret {
		return ErrReconnectSecretError
	}
	if player.socket != nil && player.socket.cwrite != nil {
		player.socket.cwrite, socket.cwrite = socket.cwrite, player.socket.cwrite
	}
	player.replace(socket)
	socket.Set(player)
	atomic.AddInt32(&this.length, 1)
	return nil
}

// Authentication 身份认证
// secret 断线重连需要使用的秘钥,需要使用协议发送给客户端
func (this *Socket) Authentication(uuid string, data interface{}) (secret string, err error) {
	if this.Authenticated() {
		return "", ErrAuthDataExist
	}
	if Options.SocketReconnectTime == 0 {
		this.Set(data)
		return
	}

	p := &Player{uuid: uuid, data: data, socket: this}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	players := this.cosnet.Players
	if v, loaded := players.dict.LoadOrStore(uuid, p); loaded {
		var ok bool
		if p, ok = v.(*Player); !ok {
			return "", ErrAuthDataIllegal
		}
		p.mutex.Lock()
		defer p.mutex.Unlock()
		p.replace(this)
	} else {
		p.secret = strconv.FormatInt(time.Now().Unix(), 32)
		atomic.AddInt32(&players.length, 1)
	}

	this.Set(p)
	secret = strings.Join([]string{p.secret, uuid}, "x")
	this.emit(EventTypeAuthentication)
	return
}

// Reconnect 断线重连,secret为Authentication时产生的
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
	if err := this.cosnet.Players.reconnect(this, secret[index+1:], secret[0:index]); err != nil {
		return err
	}
	this.emit(EventTypeReconnected)
	return nil
}

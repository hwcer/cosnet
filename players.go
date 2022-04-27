package cosnet

import "sync"

type iPlayer interface {
	ID() string //用户唯一ID
}

type player struct {
	Data   iPlayer //用户数据
	Socket *Socket
	Mutex  sync.Mutex
}

type players struct {
	dict   *sync.Map
	cosnet *Cosnet
}

func (this *player) ID() string {
	return this.Data.ID()
}

func (this *players) remove(socket *Socket) {
	//p.Mutex.Lock()
	//defer p.Mutex.Unlock()
	//if socket.status == StatusTypeRemoved || p.Socket.Id() == socket.Id() {
	//	this.players.Delete(p.ID())
	//	if p.Socket.Id() != socket.Id() {
	//		p.Socket.Close()
	//	}
	//}
}

//Authentication 登录
func (this *Cosnet) Authentication(socket *Socket, data iPlayer) {
	if socket.Authenticated() {
		return
	}
	if Options.SocketReconnectTime == 0 {
		socket.Set(data)
		return
	}

	id := data.ID()
	p := &player{Data: data, Socket: socket}
	p.Mutex.Lock()
	defer p.Mutex.Unlock()
	if v, loaded := this.players.dict.LoadOrStore(id, p); loaded {
		var ok bool
		if p, ok = v.(*player); !ok {
			return
		}
		p.Mutex.Lock()
		defer p.Mutex.Unlock()
	}

	this.Emit(EventTypeAuthentication, socket)
}

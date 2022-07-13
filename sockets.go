package cosnet

import (
	"github.com/hwcer/cosgo/smap"
)

func newSetter(id uint64, val interface{}) smap.Interface {
	d := val.(*Socket)
	d.Setter = smap.NewSetter(id, nil)
	return d
}

func newSockets() *sockets {
	s := &sockets{dict: smap.New(1024)}
	s.dict.NewSetter = newSetter
	return s
}

type sockets struct {
	dict *smap.Array
}

func (this *sockets) Len() int {
	return this.dict.Size()
}

func (this *sockets) Player(id uint64) (p *player, ok bool) {
	var s *Socket
	if s, ok = this.Socket(id); !ok {
		return
	}
	v := s.Get()
	if v != nil {
		p, ok = v.(*player)
	} else {
		ok = false
	}
	return
}

func (this *sockets) Socket(id uint64) (s *Socket, ok bool) {
	var v interface{}
	if v, ok = this.dict.Get(id); !ok {
		return
	}
	s, ok = v.(*Socket)
	return
}
func (this *sockets) Range(callback func(socket *Socket) bool) {
	this.dict.Range(func(v smap.Interface) bool {
		if s, ok := v.(*Socket); ok {
			return callback(s)
		}
		return true
	})
}

func (this *sockets) Push(socket *Socket) uint64 {
	return this.dict.Push(socket).Id()
}

func (this *sockets) remove(socket *Socket) {
	this.dict.Delete(socket.Id())
}

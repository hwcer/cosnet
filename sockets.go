package cosnet

import (
	"github.com/hwcer/cosgo/smap"
)

func NewSetter(id smap.MID, val interface{}) smap.Setter {
	d := val.(*Socket)
	d.Data = smap.NewData(id, nil)
	return d
}

func NewSockets() *Sockets {
	i := &Sockets{Array: smap.New(1024)}
	i.Array.NewSetter = NewSetter
	return i
}

type Sockets struct {
	*smap.Array
}

func (this *Sockets) Len() int {
	return this.Array.Size()
}

func (this *Sockets) Socket(id smap.MID) *Socket {
	v, ok := this.Array.Get(id)
	if !ok {
		return nil
	}
	s, _ := v.(*Socket)
	return s
}
func (this *Sockets) Player(id smap.MID) *Player {
	s := this.Socket(id)
	if s != nil {
		return s.Player()
	}
	return nil
}

func (this *Sockets) Range(callback func(socket *Socket) bool) {
	this.Array.Range(func(v smap.Setter) bool {
		if s, ok := v.(*Socket); ok {
			return callback(s)
		}
		return true
	})
}

func (this *Sockets) Create(socket *Socket) smap.Setter {
	return this.Array.Create(socket)
}

func (this *Sockets) Remove(socket *Socket) {
	if socket != nil {
		this.Array.Delete(socket.Id())
	}
}

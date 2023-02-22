package cosnet

import "sync"

var Pool = poll{Pool: sync.Pool{}}

func init() {
	Pool.Pool.New = func() interface{} {
		return &Message{}
	}
}

type poll struct {
	sync.Pool
}

func (this *poll) Acquire() *Message {
	r, _ := this.Pool.Get().(*Message)
	return r
}

func (this *poll) Release(i *Message) {
	i.Release()
	this.Pool.Put(i)
}

package message

import "sync"

var pool *sync.Pool

func init() {
	pool = &sync.Pool{}
	pool.New = func() interface{} {
		return Options.New()
	}
}

func Require() Message {
	i := pool.Get()
	return i.(Message)
}

func Release(i Message) {
	i.Release()
	pool.Put(i)
}

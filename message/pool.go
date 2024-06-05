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
	if Options.Pool {
		i := pool.Get()
		return i.(Message)
	} else {
		return Options.New()
	}

}

func Release(i Message) {
	if Options.Pool {
		i.Release()
		pool.Put(i)
	}

}

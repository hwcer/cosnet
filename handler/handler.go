package handler

import (
	"github.com/hwcer/cosnet/sockets"
	"github.com/hwcer/registry"
	"reflect"
	"sync"
	"sync/atomic"
)

const HeadSize = 6

func New() *Handler {
	i := &Handler{pool: &sync.Pool{}}
	i.pool.New = func() interface{} {
		return &Message{pool: i.pool}
	}
	opts := registry.NewOptions()
	opts.Filter = i.filter
	i.Registry = registry.New(opts)
	return i
}

type Handler struct {
	*registry.Registry
	pool   *sync.Pool
	Caller RegistryCaller //默认消息调用
	Filter RegistryFilter //服务注册过滤
}

func (this *Handler) Head() int {
	return HeadSize
}

func (this *Handler) Call(socket *sockets.Socket, msg sockets.Message) (r bool) {
	defer func() {
		if err := recover(); err != nil {
			r = socket.Errorf(err)
		}
	}()
	code, ok := msg.Code().(string)
	if !ok {
		return socket.Errorf("Service not exist")
	}
	urlPath := this.Registry.Clean(code)
	service, ok := this.Match(urlPath)
	if !ok {
		return socket.Errorf("Service not exist")
	}
	pr, fn, ok := service.Match(urlPath)
	if !ok {
		return socket.Errorf("ServiceMethod not exist")
	}
	var err error
	var reply interface{}
	if this.Caller != nil {
		reply, err = this.Caller(socket, msg, pr, fn)
	} else {
		reply, err = this.caller(socket, msg, pr, fn)
	}
	if err != nil {
		return socket.Errorf(err)
	}
	if reply != nil {
		if err = msg.Marshal(code, reply); err != nil {
			return socket.Errorf(err)
		} else {
			_ = socket.Write(msg)
		}
	} else {
		msg.Release()
	}

	return true
}

func (this *Handler) Acquire() sockets.Message {
	r, _ := this.pool.Get().(*Message)
	if r == nil || !atomic.CompareAndSwapInt32(&r.reset, 0, 1) {
		return this.Acquire()
	}
	return r
}

func (this *Handler) Register(i interface{}) error {
	s := this.Registry.Service("")
	return s.Register(i)
}

func (this *Handler) filter(s *registry.Service, pr, fn reflect.Value) bool {
	if this.Filter != nil {
		return this.Filter(s, pr, fn)
	}
	if !pr.IsValid() {
		_, ok := fn.Interface().(func(socket *sockets.Socket, msg sockets.Message) interface{})
		return ok
	}
	t := fn.Type()
	if t.NumIn() != 3 {
		return false
	}
	if t.NumOut() != 1 {
		return false
	}
	return true
}

func (this *Handler) caller(socket *sockets.Socket, msg sockets.Message, pr, fn reflect.Value) (reply interface{}, err error) {
	if !pr.IsValid() {
		f, _ := fn.Interface().(func(socket *sockets.Socket, msg sockets.Message) interface{})
		reply = f(socket, msg)
	} else if s, ok := pr.Interface().(RegistryHandle); ok {
		reply = s.Caller(socket, msg, fn)
	} else {
		ret := fn.Call([]reflect.Value{pr, reflect.ValueOf(socket), reflect.ValueOf(msg)})
		reply = ret[0].Interface()
	}
	return
}

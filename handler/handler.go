package handler

import (
	"github.com/hwcer/cosnet/sockets"
	"github.com/hwcer/registry"
	"sync"
	"sync/atomic"
)

const HeadSize = 6

func New() *Handler {
	i := &Handler{pool: &sync.Pool{}}
	i.pool.New = func() interface{} {
		return &Context{}
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

func (this *Handler) Handle(socket *sockets.Socket, i sockets.Message) (r bool) {
	defer func() {
		if err := recover(); err != nil {
			r = socket.Errorf(err)
		}
	}()

	ctx, ok := i.(*Context)
	if !ok {
		return socket.Errorf("message error")
	}
	ctx.Socket = socket
	autoRelease := true
	defer func() {
		if autoRelease {
			this.Release(ctx)
		}
	}()

	path := ctx.Path()
	urlPath := this.Registry.Clean(path)
	service, ok := this.Match(urlPath)
	if !ok {
		return socket.Errorf("Binder not exist")
	}
	node, ok := service.Match(urlPath)
	if !ok {
		return socket.Errorf("ServiceMethod not exist")
	}
	var err error
	var reply interface{}
	if this.Caller != nil {
		reply, err = this.Caller(ctx, node)
	} else {
		reply, err = this.caller(ctx, node)
	}
	if err != nil {
		return socket.Errorf(err)
	}
	if reply != nil {
		if err = ctx.Marshal(path, reply); err != nil {
			return socket.Errorf(err)
		} else {
			autoRelease = false
			_ = socket.Write(ctx)
		}
	}
	return true
}

func (this *Handler) Acquire() sockets.Message {
	r, _ := this.pool.Get().(*Context)
	if r == nil || !atomic.CompareAndSwapInt32(&r.pool, 0, 1) {
		return this.Acquire()
	}
	return r
}

func (this *Handler) Release(i sockets.Message) {
	ctx, ok := i.(*Context)
	if !ok {
		return
	}
	if atomic.CompareAndSwapInt32(&ctx.pool, 1, 0) {
		ctx.Socket = nil
		this.pool.Put(ctx)
	}
}

func (this *Handler) Register(i interface{}) error {
	return this.Registry.Register(i)
}

func (this *Handler) filter(service *registry.Service, node *registry.Node) bool {
	if this.Filter != nil {
		return this.Filter(service, node)
	}
	if node.IsFunc() {
		_, ok := node.Method().(func(ctx *Context) interface{})
		return ok
	}
	fn := node.Value()
	t := fn.Type()
	if t.NumIn() != 2 {
		return false
	}
	if t.NumOut() != 1 {
		return false
	}
	return true
}

func (this *Handler) caller(ctx *Context, node *registry.Node) (reply interface{}, err error) {
	if node.IsFunc() {
		f, _ := node.Method().(func(ctx *Context) interface{})
		reply = f(ctx)
	} else if s, ok := node.Binder().(RegistryHandle); ok {
		reply = s.Caller(ctx, node)
	} else {
		ret := node.Call(ctx)
		reply = ret[0].Interface()
	}
	return
}

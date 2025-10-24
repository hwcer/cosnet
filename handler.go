package cosnet

import (
	"reflect"

	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/logger"
)

type handleCaller interface {
	Caller(node *registry.Node, c *Context) any
}
type HandlerFilter func(node *registry.Node) bool
type HandlerCaller func(node *registry.Node, c *Context) any
type HandlerSerialize func(c *Context, reply any) ([]byte, error)

type Handler struct {
	filter    HandlerFilter
	caller    HandlerCaller
	serialize HandlerSerialize //消息序列化封装,仅仅针对确认包
}

func (this *Handler) SetCaller(caller func(node *registry.Node, c *Context) any) {
	this.caller = caller
}

func (this *Handler) SetFilter(filter func(node *registry.Node) bool) {
	this.filter = filter
}
func (this *Handler) SetSerialize(serialize func(c *Context, reply any) ([]byte, error)) {
	this.serialize = serialize
}

func (this *Handler) Filter(node *registry.Node) bool {
	if this.filter != nil {
		return this.filter(node)
	}
	if node.IsFunc() {
		_, ok := node.Method().(func(*Context) any)
		return ok
	} else if node.IsMethod() {
		t := node.Value().Type()
		if t.NumIn() != 2 || t.NumOut() != 1 {
			return false
		}
		return true
	} else {
		if _, ok := node.Binder().(handleCaller); !ok {
			v := reflect.Indirect(reflect.ValueOf(node.Binder()))
			logger.Debug("[%v]未正确实现Caller方法,会影响程序性能", v.Type().String())
		}
		return true
	}
}

func (this *Handler) handle(node *registry.Node, c *Context) (reply any) {
	if this.caller != nil {
		return this.caller(node, c)
	}
	if node.IsFunc() {
		f := node.Method().(func(*Context) any)
		reply = f(c)
	} else if s, ok := node.Binder().(handleCaller); ok {
		reply = s.Caller(node, c)
	} else {
		r := node.Call(c)
		reply = r[0].Interface()
	}
	return
}

func (this *Handler) write(c *Context, reply any) (err error) {
	p, must := c.Message.Confirm()
	if !must {
		return nil
	}
	switch v := reply.(type) {
	case []byte:
		c.Send(p, v)
	case *[]byte:
		c.Send(p, *v)
	default:
		var data []byte
		if this.serialize != nil {
			data, err = this.serialize(c, reply)
		} else {
			data, err = this.defaultSerialize(c, reply)
		}
		if err == nil {
			c.Send(p, data)
		}
	}
	return
}

func (this *Handler) defaultSerialize(c *Context, reply any) ([]byte, error) {
	b := c.Message.Binder()
	return b.Marshal(reply)
}

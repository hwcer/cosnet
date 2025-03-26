package cosnet

import (
	"github.com/hwcer/cosgo/logger"
	"github.com/hwcer/cosgo/registry"
	"reflect"
)

type handleCaller interface {
	Caller(node *registry.Node, c *Context) error
}
type HandlerFilter func(node *registry.Node) bool
type HandlerCaller func(node *registry.Node, c *Context) error

//type HandlerSerialize func(c *Context, reply any) any

type Handler struct {
	filter HandlerFilter
	caller HandlerCaller
	//serialize HandlerSerialize
}

func (this *Handler) Use(src any) {
	if v, ok := src.(HandlerFilter); ok {
		this.filter = v
	}
	if v, ok := src.(HandlerCaller); ok {
		this.caller = v
	}
	//if v, ok := src.(HandlerSerialize); ok {
	//	this.serialize = v
	//}
}

func (this *Handler) Filter(node *registry.Node) bool {
	if this.filter != nil {
		return this.filter(node)
	}
	if node.IsFunc() {
		_, ok := node.Method().(func(*Context) error)
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

func (this *Handler) Caller(node *registry.Node, c *Context) (err error) {
	if this.caller != nil {
		return this.caller(node, c)
	}
	if node.IsFunc() {
		f := node.Method().(func(*Context) error)
		err = f(c)
	} else if s, ok := node.Binder().(handleCaller); ok {
		err = s.Caller(node, c)
	} else {
		r := node.Call(c)
		err = r[0].Interface().(error)
	}
	//if this.serialize != nil {
	//	err = this.serialize(c, reply)
	//}
	return
}

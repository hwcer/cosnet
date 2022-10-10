package cosnet

import (
	"github.com/hwcer/cosgo/message"
	"github.com/hwcer/logger"
	"github.com/hwcer/registry"
	"reflect"
)

type handleCaller interface {
	Caller(node *registry.Node, c *Context) interface{}
}
type HandlerCaller func(node *registry.Node, c *Context) (interface{}, error)
type HandlerSerialize func(c *Context, reply interface{}) (newReply interface{})

type Handler struct {
	caller    HandlerCaller
	serialize HandlerSerialize
}

func (this *Handler) Use(src interface{}) {
	if v, ok := src.(HandlerCaller); ok {
		this.caller = v
	}
	if v, ok := src.(HandlerSerialize); ok {
		this.serialize = v
	}
}

func (this *Handler) Filter(node *registry.Node) bool {
	if node.IsFunc() {
		_, ok := node.Method().(func(*Context) interface{})
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
func (this *Handler) handle() {}

func (this *Handler) Caller(node *registry.Node, c *Context) (reply interface{}, err error) {
	if this.caller != nil {
		return this.caller(node, c)
	}
	if node.IsFunc() {
		f := node.Method().(func(*Context) interface{})
		reply = f(c)
	} else if s, ok := node.Method().(handleCaller); ok {
		reply = s.Caller(node, c)
	} else {
		r := node.Call(c)
		reply = r[0].Interface()
	}
	return
}

func (this *Handler) Serialize(c *Context, reply interface{}) error {
	if reply != nil {

	}

	if reply == nil {
		c.Socket.Agents.Release(c.Message)
		return nil
	} else if this.serialize != nil {
		return this.serialize(c, reply)
	} else {
		return Serialize(c, reply)
	}

}

func Serialize(c *Context, reply interface{}) (err error) {
	switch v := reply.(type) {
	case []byte:
		return c.Reply(0, v)
	default:
		msg := message.Parse(reply)
		return c.Reply(int32(msg.Code), []byte(msg.Data))
	}
}

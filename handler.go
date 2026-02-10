package cosnet

import (
	"reflect"

	"github.com/hwcer/cosgo/registry"
	"github.com/hwcer/cosnet/message"
	"github.com/hwcer/logger"
)

// handleCaller 处理器调用接口，用于自定义调用逻辑
type handleCaller interface {
	// Caller 调用方法，处理请求并返回响应
	// 参数:
	//   node: 注册的节点
	//   c: 上下文
	// 返回值:
	//   响应数据
	Caller(node *registry.Node, c *Context) any
}

// HandlerFilter 处理器过滤器类型，用于过滤处理器
// 参数:
//
//	node: 注册的节点
//
// 返回值:
//
//	是否通过过滤
type HandlerFilter func(node *registry.Node) bool

// HandlerCaller 处理器调用函数类型，用于自定义调用逻辑
// 参数:
//
//	node: 注册的节点
//	c: 上下文
//
// 返回值:
//
//	响应数据
type HandlerCaller func(node *registry.Node, c *Context) any

// HandlerSerialize 处理器序列化函数类型，用于自定义序列化逻辑
// 参数:
//
//	c: 上下文
//	reply: 响应数据
//
// 返回值:
//
//	序列化后的字节数组和错误信息
type HandlerSerialize func(c *Context, reply any) ([]byte, error)

// Handler 消息处理器，用于处理消息和生成响应
type Handler struct {
	filter    HandlerFilter    // 处理器过滤器
	caller    HandlerCaller    // 处理器调用函数
	serialize HandlerSerialize // 消息序列化封装,仅仅针对确认包
}

// SetCaller 设置处理器调用函数
// 参数:
//
//	caller: 处理器调用函数
func (this *Handler) SetCaller(caller func(node *registry.Node, c *Context) any) {
	this.caller = caller
}

// SetFilter 设置处理器过滤器
// 参数:
//
//	filter: 处理器过滤器
func (this *Handler) SetFilter(filter func(node *registry.Node) bool) {
	this.filter = filter
}

// SetSerialize 设置序列化函数
// 参数:
//
//	serialize: 序列化函数
func (this *Handler) SetSerialize(serialize func(c *Context, reply any) ([]byte, error)) {
	this.serialize = serialize
}

// Filter 过滤处理器
// 参数:
//
//	node: 注册的节点
//
// 返回值:
//
//	是否通过过滤
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

// handle 处理消息
// 参数:
//
//	node: 注册的节点
//	c: 上下文
//
// 返回值:
//
//	响应数据
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

// reply 写入响应
// 参数:
//
//	c: 上下文
//	reply: 响应数据
//
// 返回值:
//
//	错误信息
func (this *Handler) reply(c *Context, reply any) (err error) {
	flag := c.Message.Flag()
	if !flag.Has(message.FlagNeedACK) {
		return
	}
	replyFlag := message.FlagIsACK
	replyIndex := c.Message.Index()
	replyConfirm := c.Message.Confirm()

	switch v := reply.(type) {
	case []byte:
		c.Socket.Send(replyFlag, replyIndex, replyConfirm, v)
	case *[]byte:
		c.Socket.Send(replyFlag, replyIndex, replyConfirm, *v)
	default:
		var data []byte
		if this.serialize != nil {
			data, err = this.serialize(c, reply)
		} else {
			data, err = this.defaultSerialize(c, reply)
		}
		if err == nil {
			c.Socket.Send(replyFlag, replyIndex, replyConfirm, data)
		}
	}
	return
}

// defaultSerialize 默认序列化方法
// 参数:
//
//	c: 上下文
//	reply: 响应数据
//
// 返回值:
//
//	序列化后的字节数组和错误信息
func (this *Handler) defaultSerialize(c *Context, reply any) ([]byte, error) {
	b := c.Message.Binder()
	return b.Marshal(reply)
}

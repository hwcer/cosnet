package cosnet

import (
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosnet/message"
)

// Context 上下文，封装了Socket和Message，用于处理请求和响应
type Context struct {
	*Socket                 // 网络连接
	Message message.Message // 当前处理的消息
}

// Path 获取消息的路径和查询参数
// 返回值:
//
//	第一个返回值: 消息路径
//	第二个返回值: 查询参数
//	第三个返回值: 错误信息
func (this *Context) Path() (string, string, error) {
	return this.Message.Path()
}

// Bind 将消息体绑定到指定的结构体
// 参数:
//
//	i: 要绑定的结构体指针
//
// 返回值:
//
//	错误信息，如果绑定成功则为nil
func (this *Context) Bind(i any) error {
	return this.Message.Unmarshal(i)
}

// Send 发送消息到客户端
// 参数:
//   path: 消息路径
//   data: 消息数据
//func (this *Context) Send(path string, data any) {
//	i := this.Message.Index()
//	this.Socket.Send(i, path, data)
//}

// Write 直接写入消息到客户端
// 参数:
//
//	m: 要发送的消息
func (this *Context) Write(m message.Message) {
	this.Socket.Write(m)
}

// Error 使用当前路径向客户端写入一个默认错误码的信息
// 参数:
//
//	err: 错误信息
func (this *Context) Error(err any) {
	Errorf(this.Socket, err)
}

// Errorf 使用当前路径向客户端写入一个特定错误码的信息
// 参数:
//
//	format: 错误格式
//	args: 错误参数
func (this *Context) Errorf(format any, args ...any) {
	Errorf(this.Socket, format, args...)
}

// Accept 获取消息的绑定器
// 返回值:
//
//	消息的绑定器，用于序列化和反序列化
func (this *Context) Accept() binder.Binder {
	return this.Message.Binder()
}

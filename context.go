package cosnet

import (
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosnet/message"
)

// Context 封装了 Socket 和 Message，用于处理请求和响应。
type Context struct {
	*Socket                 // 网络连接
	Message message.Message // 当前处理的消息
}

// Path 获取消息的路径和查询参数。
// 返回值:
//   - string: 消息路径
//   - string: 查询参数
//   - error: 错误信息
func (this *Context) Path() (string, string, error) {
	return this.Message.Path()
}

// Bind 将消息体绑定到指定的结构体。
// 参数 i: 要绑定的结构体指针。
// 返回值: 错误信息，如果绑定成功则为 nil。
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

// Write 直接写入消息到客户端。
// 参数 m: 要发送的消息。
func (this *Context) Write(m message.Message) {
	this.Socket.Write(m)
}

// Accept 获取消息的绑定器。
// 返回值: 消息的绑定器，用于序列化和反序列化。
func (this *Context) Accept() binder.Binder {
	return this.Message.Binder()
}

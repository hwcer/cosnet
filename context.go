package cosnet

import (
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosnet/message"
)

// Context 封装了 Socket 和 Message，用于处理请求和响应。
type Context struct {
	*Socket                 // 网络连接
	flag    message.Flag    // 出站(确认包)flag，零值即无附加；与入站 Message.Flag() 相互独立
	Message message.Message // 当前处理的消息
}

// Flag 出站 flag：无参取值，传参设置（如 FlagCompressed/FlagEncrypted）。
// flag 描述的是包自身的属性，故入站与出站各自独立：
// 本方法只作用于确认包，入站包的 flag 请读 Message.Flag()，两者互不影响。
// FlagConfirm 与心跳标记由 Handler.reply 补齐，业务层无需也无法抹掉。
func (this *Context) Flag(set ...message.Flag) message.Flag {
	if len(set) > 0 {
		this.flag = set[0]
	}
	return this.flag
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

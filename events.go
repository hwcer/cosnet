package cosnet

// EventType 定义事件类型。
type EventType uint8

// 事件类型常量定义。
const (
	EventTypeError          EventType = iota + 1 // 系统级别错误事件,参数: Socket||nil,错误信息
	EventTypeHeartbeat                           // 心跳事件,参数:Socket,心跳计数增量
	EventTypeMessage                             // 所有未注册的消息事件,参数:Socket,消息内容
	EventTypeConnected                           // 连接成功事件,参数:Socket,nil
	EventTypeReconnected                         // 断线重连事件,参数:Socket,nil
	EventTypeDisconnect                          // 断开连接事件,参数:Socket,nil
	EventTypeAuthentication                      // 身份认证事件,参数:Socket,是否重连
	EventTypeReplaced                            // 被顶号事件,参数:Socket,新Socket ip
)

// EventsFunc 定义事件处理函数类型。
// 参数:
//   - *Socket: 触发事件的 Socket
//   - any: 事件附加数据
type EventsFunc func(*Socket, any)

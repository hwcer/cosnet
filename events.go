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
	EventTypeReplaced                            // 顶号协商事件,参数:Socket,*Replaced
)

// Replaced 顶号协商事件(EventTypeReplaced)的参数。
//
// 语义是"**你被顶号了**,还剩 Timeout 秒",不是"这条连接已经废了":
// 收到它的连接进入存活期,期间只收不发,倒计时结束才真正断开(见 Socket.Replaced)。
// 新端是立刻接管还是等它下线,由上层策略决定,这里看不出来。
type Replaced struct {
	Address string // 请求顶号的客户端 IP
	Timeout int32  // 剩余存活秒数,归零后本连接被断开(两种顶号策略下都一样)
}

// EventsFunc 定义事件处理函数类型。
// 参数:
//   - *Socket: 触发事件的 Socket
//   - any: 事件附加数据
type EventsFunc func(*Socket, any)

package cosnet

import (
	"github.com/hwcer/cosgo/values"
	"github.com/hwcer/logger"
)

// EventType 事件类型枚举
type EventType uint8

const (
	// EventTypeError 系统级别错误事件
	EventTypeError          EventType = iota + 1
	// EventTypeMessage 所有未注册的消息事件
	EventTypeMessage
	// EventTypeConnected 连接成功事件
	EventTypeConnected
	// EventTypeReconnected 断线重连事件
	EventTypeReconnected
	// EventTypeAuthentication 身份认证事件，参数：是否重连
	EventTypeAuthentication
	// EventTypeDisconnect 断开连接事件
	EventTypeDisconnect
	// EventTypeReplaced 被顶号事件
	EventTypeReplaced
)

// EventsFunc 事件处理函数类型
// 参数:
//   第一个参数: 触发事件的Socket
//   第二个参数: 事件附加数据
type EventsFunc func(*Socket, any)

// On 注册事件处理函数
// 参数:
//   e: 事件类型
//   f: 事件处理函数
func On(e EventType, f EventsFunc) {
	emitter[e] = append(emitter[e], f)
}

// Emit 触发事件
// 参数:
//   e: 事件类型
//   s: 触发事件的Socket
//   attach: 事件附加数据
func Emit(e EventType, s *Socket, attach ...any) {
	var v any
	if len(attach) > 0 {
		v = attach[0]
	}
	for _, f := range emitter[e] {
		f(s, v)
	}
}

// Errorf 抛出一个异常事件
// 参数:
//   s: 触发错误的Socket
//   format: 错误格式
//   args: 错误参数
func Errorf(s *Socket, format any, args ...any) {
	defer func() {
		if e := recover(); e != nil {
			logger.Error(e)
		}
	}()
	var err any
	if len(args) > 0 {
		err = values.Sprintf(format, args...)
	} else {
		err = format
	}
	Emit(EventTypeError, s, err)
}

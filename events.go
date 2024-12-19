package cosnet

import (
	"github.com/hwcer/cosgo/logger"
)

// EventType 事件类型
type EventType uint8

const (
	EventTypeError          EventType = iota + 1 //系统级别错误
	EventTypeMessage                             //所有未注册的消息
	EventTypeConnected                           //连接成功
	EventTypeAuthentication                      //身份认证
	EventTypeDisconnect                          //断开连接
)

type EventsFunc func(*Socket, any)

func On(e EventType, f EventsFunc) {
	emitter[e] = append(emitter[e], f)
}

func Emit(e EventType, s *Socket, attach ...any) {
	var v any
	if len(attach) > 0 {
		v = attach[0]
	}
	for _, f := range emitter[e] {
		f(s, v)
	}
}

// Errorf 抛出一个异常
func Errorf(s *Socket, format any, args ...any) {
	defer func() {
		if e := recover(); e != nil {
			logger.Error(e)
		}
	}()
	err := logger.Sprintf(format, args...)
	Emit(EventTypeError, s, err)
}

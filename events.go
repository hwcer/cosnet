package cosnet

import (
	"github.com/hwcer/cosgo/logger"
	"github.com/hwcer/cosgo/values"
)

// EventType 事件类型
type EventType uint8

const (
	EventTypeError          EventType = iota + 1 //系统级别错误
	EventTypeMessage                             //所有未注册的消息
	EventTypeConnected                           //连接成功
	EventTypeReconnected                         //断线重连
	EventTypeAuthentication                      //身份认证
	EventTypeDisconnect                          //断开连接
	EventTypeReplaced                            //被顶号
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
	err := values.Sprintf(format, args...)
	Emit(EventTypeError, s, err)
}

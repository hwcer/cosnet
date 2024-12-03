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
	EventTypeReconnected                         //重登录
	EventTypeReplaced                            //被顶号
	EventTypeReleased                            //释放所有信息
)

type EventsFunc func(*Socket, any) bool

func On(e EventType, f EventsFunc) {
	emitter[e] = append(emitter[e], f)
}

func Emit(e EventType, s *Socket, attach ...interface{}) (r bool) {
	var v interface{}
	if len(attach) > 0 {
		v = attach[0]
	}
	for _, f := range emitter[e] {
		if r = f(s, v); !r {
			return r
		}
	}
	return true
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

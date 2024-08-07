package cosnet

import (
	"github.com/hwcer/logger"
)

// EventType 事件类型
type EventType uint8

const (
	EventTypeError       EventType = iota + 1 //系统级别错误
	EventTypeMessage                          //所有未注册的消息
	EventTypeHeartbeat                        //心跳事件
	EventTypeConnected                        //连接成功
	EventTypeDisconnect                       //断开连接
	EventTypeReconnected                      //重登录
	EventTypeReplaced                         //被顶号
	EventTypeDestroyed                        //销毁所有信息

)

type EventsFunc func(*Socket, any) bool

func (this *Server) On(e EventType, f EventsFunc) {
	this.events[e] = append(this.events[e], f)
}

func (this *Server) Emit(e EventType, s *Socket, attach ...interface{}) (r bool) {
	var v interface{}
	if len(attach) > 0 {
		v = attach[0]
	}
	for _, f := range this.events[e] {
		if r = f(s, v); !r {
			return r
		}
	}
	return true
}

// Errorf 抛出一个异常
func (this *Server) Errorf(s *Socket, format any, args ...any) {
	defer func() {
		if e := recover(); e != nil {
			logger.Error(e)
		}
	}()
	err := logger.Sprintf(format, args...)
	this.Emit(EventTypeError, s, err)
}

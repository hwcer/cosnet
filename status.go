package cosnet

import "sync/atomic"

//type StatusType int32

const (
	StatusTypeConnect int32 = iota
	StatusTypeDisconnect
	StatusTypeDestroyed
)

//func NewStatus() *Status {
//	return &Status{}
//}

type Status struct {
	//mutex     sync.mutex
	status    int32  //0-正常，1-断开，2-强制关闭
	closing   bool   //强制关闭中
	heartbeat uint32 //heartbeat >=timeout 时被标记为超时
}

//func (this *Status) Has(ss ...StatusType) bool {
//	for _, s := range ss {
//		if this.status == s {
//			return true
//		}
//	}
//	return false
//}

// Close 强制关闭,无法重登陆
func (this *Status) Close() bool {
	if this.status == StatusTypeDestroyed || this.closing {
		return false
	}
	this.closing = true
	return true
}

// Disabled 是否已经关闭接收消息
func (this *Status) Disabled() bool {
	return this.closing || this.status == StatusTypeDestroyed
}

// Destroy 彻底销毁,移除资源
func (this *Status) Destroy() (r bool) {
	if this.status != StatusTypeDestroyed {
		r = true
		this.status = StatusTypeDestroyed
	}
	return
}

// Reconnect 重新登录
func (this *Status) Reconnect() bool {
	return atomic.CompareAndSwapInt32(&this.status, StatusTypeDisconnect, StatusTypeConnect)
}

// Disconnect 掉线,包含网络超时，网络错误
func (this *Status) Disconnect() bool {
	return atomic.CompareAndSwapInt32(&this.status, StatusTypeConnect, StatusTypeDisconnect)
}

// KeepAlive 任何行为都清空heartbeat
func (this *Status) KeepAlive() {
	this.heartbeat = 0
}

func (this *Status) Heartbeat() uint32 {
	this.heartbeat += Options.SocketHeartbeat
	return this.heartbeat
}

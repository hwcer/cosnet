package cosnet

import "sync"

type StatusType int8

const (
	StatusTypeConnect StatusType = iota
	StatusTypeClosing            //强制关闭中
	StatusTypeClosed             //已经关闭
	StatusTypeDisconnect
	StatusTypeDestroyed
)

func NewStatus() *Status {
	return &Status{mutex: sync.Mutex{}}
}

type Status struct {
	mutex     sync.Mutex
	status    StatusType //0-正常，1-断开，2-强制关闭
	heartbeat uint32     //heartbeat >=timeout 时被标记为超时
}

func (this *Status) Has(ss ...StatusType) bool {
	for _, s := range ss {
		if this.status == s {
			return true
		}
	}
	return false
}

// Close 强制关闭,无法重登陆
func (this *Status) Close(f func(bool)) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	var r bool
	if this.status == StatusTypeConnect {
		r = true
		this.status = StatusTypeClosing
	}
	if f != nil {
		f(r)
	}
}

// Disconnect 掉线,包含网络超时，网络错误
func (this *Status) Disconnect(f func(bool)) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	var r bool
	if this.status == StatusTypeConnect {
		r = true
		this.status = StatusTypeDisconnect
	} else if this.status == StatusTypeClosing {
		r = true
		this.status = StatusTypeClosed
	}

	if f != nil {
		f(r)
	}
}

// Destroy 彻底销毁,移除资源
func (this *Status) Destroy(f func(bool)) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	var r bool
	if this.status != StatusTypeDestroyed {
		r = true
		this.status = StatusTypeDestroyed
	}
	if f != nil {
		f(r)
	}
}

// Reconnect 重新登录
func (this *Status) Reconnect(f func(bool)) {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	var r bool
	if this.status == StatusTypeDisconnect {
		r = true
		this.status = StatusTypeConnect
	}
	if f != nil {
		f(r)
	}
}

// KeepAlive 任何行为都清空heartbeat
func (this *Status) KeepAlive() {
	this.heartbeat = 0
}

func (this *Status) Heartbeat() uint32 {
	this.heartbeat += Options.SocketHeartbeat
	return this.heartbeat
}

package cosnet

import (
	"context"
	"github.com/hwcer/logger"
	"time"
)

func stop() {
	for _, l := range instance {
		_ = l.Close()
	}
}

// heartbeat 启动协程定时清理无效用户
//
//	Options.SocketHeartbeat == 0 不启动计时器，由业务层直接调用Heartbeat
func heartbeat(ctx context.Context) {
	if Options.Heartbeat == 0 {
		logger.Debug("cosnet heartbeat not start,Because Options.Heartbeat is 0")
		return
	}
	t := time.Second * time.Duration(Options.Heartbeat)
	ticker := time.NewTimer(t)
	defer ticker.Stop()
	defer stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			Heartbeat(Options.Heartbeat)
			ticker.Reset(t)
		}
	}
}

func Heartbeat(v int32) {
	Range(func(socket *Socket) bool {
		socket.doHeartbeat(v)
		return true
	})
}

// doHeartbeat 每一次Heartbeat() heartbeat计数加1
func (sock *Socket) doHeartbeat(v int32) {
	if Options.SocketConnectTime > 0 && sock.Heartbeat(v) > Options.SocketConnectTime {
		sock.disconnect()
	}
}

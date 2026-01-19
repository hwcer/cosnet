package cosnet

import (
	"context"
	"time"

	"github.com/hwcer/logger"
)

// stop 停止所有监听器
func stop() {
	for _, l := range instance {
		_ = l.Close()
	}
}

// heartbeat 启动协程定时清理无效用户
// 参数:
//   ctx: 上下文，用于控制协程退出
// 注意:
//   Options.Heartbeat == 0 时不启动计时器，由业务层直接调用Heartbeat
func heartbeat(ctx context.Context) {
	if Options.Heartbeat == 0 {
		logger.Debug("cosnet heartbeat not start,Because Options.Heartbeat is 0")
		return
	}
	// 计算心跳间隔
	t := time.Second * time.Duration(Options.Heartbeat)
	ticker := time.NewTimer(t)
	defer ticker.Stop()
	defer stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 执行心跳检查
			Heartbeat(Options.Heartbeat)
			// 重置计时器
			ticker.Reset(t)
		}
	}
}

// Heartbeat 对所有连接执行心跳检查
// 参数:
//   v: 心跳计数增量
func Heartbeat(v int32) {
	Range(func(socket *Socket) bool {
		socket.doHeartbeat(v)
		return true
	})
}

// doHeartbeat 执行单个连接的心跳检查
// 每一次Heartbeat()调用，heartbeat计数加1
// 参数:
//   v: 心跳计数增量
func (sock *Socket) doHeartbeat(v int32) {
	// 如果设置了连接超时时间，并且心跳计数超过了超时时间，则断开连接
	if Options.SocketConnectTime > 0 && sock.Heartbeat(v) > Options.SocketConnectTime {
		sock.disconnect()
	}
}

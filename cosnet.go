package cosnet

import (
	"context"
	"errors"
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/cosnet/handler"
	"github.com/hwcer/cosnet/sockets"
	"net"
	"strings"
)

func New(ctx context.Context, h sockets.Handler) *Cosnet {
	if h == nil {
		h = handler.New()
	}
	i := &Cosnet{
		Agents: sockets.New(ctx, h),
	}
	return i
}

// Cosnet socket管理器
type Cosnet struct {
	*sockets.Agents
}

// Listen 启动柜服务器,监听address
func (this *Cosnet) Listen(address string) (listener net.Listener, err error) {
	addr := utils.NewAddress(address)
	if addr.Scheme == "" {
		return nil, errors.New("address scheme empty")
	}
	switch strings.ToLower(addr.Scheme) {
	case "tcp":
		listener, err = NewTcpServer(this.Agents, addr.String())
	//case "udp":
	//TODO
	default:
		err = errors.New("address scheme unknown")
	}
	return
}

// Connect 连接服务器address
func (this *Cosnet) Connect(address string) (socket *sockets.Socket, err error) {
	return cliConnect(this.Agents, address)
}

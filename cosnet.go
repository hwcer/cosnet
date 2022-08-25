package cosnet

import (
	"context"
	"errors"
	"github.com/hwcer/cosgo/utils"
	"github.com/hwcer/cosnet/handler"
	"github.com/hwcer/cosnet/sockets"
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
func (this *Cosnet) Listen(address string) (listener interface{}, err error) {
	addr := utils.NewAddress(address)
	if addr.Scheme == "" {
		return nil, errors.New("address scheme empty")
	}
	network := strings.ToLower(addr.Scheme)
	switch network {
	case "tcp", "tcp4", "tcp6":
		listener, err = this.NewTcpServer(network, addr.String())
	case "udp", "udp4", "udp6":
		listener, err = this.NewUdpServer(network, addr.String())
	//case "unix", "unixgram", "unixpacket":
	default:
		err = errors.New("address scheme unknown")
	}
	return
}

// Connect 连接服务器address
func (this *Cosnet) Connect(address string) (socket *sockets.Socket, err error) {
	return cliConnect(this.Agents, address)
}

package cosnet

import (
	"context"
	"github.com/hwcer/cosgo/storage/cache"
	"github.com/hwcer/cosgo/utils"
	"net"
	"sync"
	"time"
)

func newSocketSetter(id uint64, val interface{}) cache.Dataset {
	dataset := val.(cache.Dataset)
	dataset.Reset(id, nil)
	return dataset
}

func New() *Cosnet {
	cosnet := &Cosnet{
		scc:      utils.NewSCC(nil),
		handle:   make(map[uint16]HandlerFunc),
		listener: make(map[EventType][]EventsFunc),
	}
	cosnet.sockets = cache.New(1024)
	cosnet.sockets.NewSetter = newSocketSetter
	cosnet.players = &players{cosnet: cosnet, dict: new(sync.Map)}
	cosnet.scc.CGO(cosnet.heartbeat)
	return cosnet
}

//Cosnet socket管理器
type Cosnet struct {
	scc      *utils.SCC
	handle   map[uint16]HandlerFunc     //注册的消息处理器
	sockets  *cache.Cache               //存储socket
	players  *players                   //存储用户登录信息
	servers  []*Server                  //全局关闭时需要关闭的服务
	listener map[EventType][]EventsFunc //监听事件
	Handler  HandlerFunc                //默认消息处理,handle中未明确注册的消息一律进入到这里
}

func (this *Cosnet) create(socket *Socket) {
	this.sockets.Push(socket)
	this.Emit(EventTypeConnected, socket)
}

//remove 移除Socket
func (this *Cosnet) remove(socket *Socket) {
	this.Emit(EventTypeDisconnect, socket)
	this.sockets.Delete(socket.Id())
	if Options.SocketReconnectTime > 0 && socket.Authenticated() {
		go func() {
			this.players.remove(socket)
		}()
	}
}

//heartbeat 启动协程定时清理无效用户
func (this *Cosnet) heartbeat(ctx context.Context) {
	t := time.Millisecond * time.Duration(Options.SocketHeartbeat)
	ticker := time.NewTimer(t)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			utils.Try(this.doHeartbeat)
			ticker.Reset(t)
		}
	}
}
func (this *Cosnet) doHeartbeat() {
	this.sockets.Range(func(v cache.Dataset) bool {
		socket := v.(*Socket)
		socket.Heartbeat()
		this.Emit(EventTypeHeartbeat, socket)
		return true
	})
}

func (this *Cosnet) Close(timeout time.Duration) error {
	if !this.scc.Cancel() {
		return nil
	}
	for _, s := range this.servers {
		_ = s.Close()
	}
	this.servers = nil
	return this.scc.Wait(timeout)
}

//New 创建新socket并自动加入到Agents管理器
func (this *Cosnet) New(conn net.Conn, netType NetType) (sock *Socket, err error) {
	sock = &Socket{conn: conn, cosnet: this, netType: netType}
	sock.cwrite = make(chan *Message, Options.WriteChanSize)
	sock.Data = *cache.NewData()
	err = sock.start()
	if err == nil {
		this.create(sock)
	}
	return
}

//Socket 通过SOCKETID获取SOCKET
func (this *Cosnet) Socket(id uint64) (*Socket, bool) {
	if v, ok := this.sockets.Get(id); !ok {
		return nil, false
	} else if v2, ok2 := v.(*Socket); ok2 {
		return v2, true
	} else {
		return nil, false
	}
}

//Broadcast 广播,filter 过滤函数，如果不为nil且返回false则不对当期socket进行发送消息
func (this *Cosnet) Broadcast(msg *Message, filter func(*Socket) bool) {
	this.sockets.Range(func(v cache.Dataset) bool {
		sock := v.(*Socket)
		if filter == nil || filter(sock) {
			sock.Write(msg)
		}
		return true
	})
}

//Listen 启动柜服务器,监听address
func (this *Cosnet) Listen(address string) (server *Server, err error) {
	server, err = NewServer(this, address)
	if err == nil && server != nil {
		this.servers = append(this.servers, server)
		this.scc.GO(server.Start)
	}
	return
}

//Connect 连接服务器address
func (this *Cosnet) Connect(address string) (socket *Socket, err error) {
	return NewConnect(this, address)
}

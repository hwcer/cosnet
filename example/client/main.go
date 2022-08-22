package main

import (
	"github.com/hwcer/cosgo/app"
	"github.com/hwcer/cosgo/logger"
	"github.com/hwcer/cosnet"
	"github.com/hwcer/cosnet/handler"
	"github.com/hwcer/cosnet/sockets"
	"github.com/spf13/pflag"
)

var server *cosnet.Cosnet
var handle *handler.Handler

func init() {
	pflag.String("address", "tcp://0.0.0.0:3000", "server address")
}

func main() {
	app.Start(&module{ModuleDefault: app.ModuleDefault{Id: "client"}})
	app.WaitForSystemExit()
}

type module struct {
	app.ModuleDefault
}

func (m *module) Start() error {
	address := app.Config.GetString("address")
	server = cosnet.New(app.SCC.Context, nil)
	_, err := server.Connect(address)
	if err != nil {
		return err
	}
	handle = server.Handler.(*handler.Handler)
	_ = handle.Register(ping)
	server.On(sockets.EventTypeError, socketError)
	server.On(sockets.EventTypeHeartbeat, socketHeartbeat)
	server.On(sockets.EventTypeConnected, socketConnected)
	server.On(sockets.EventTypeDisconnect, socketDisconnect)
	server.On(sockets.EventTypeDestroyed, socketDestroyed)
	return nil
}

func socketError(socket *sockets.Socket, err interface{}) bool {
	logger.Error("socket error:%v", err)
	return true
}
func socketHeartbeat(socket *sockets.Socket, _ interface{}) bool {
	socket.KeepAlive()
	m := &handler.Context{}
	if err := m.Marshal("ping", "hi"); err == nil {
		socket.Write(m)
	}

	return true
}

func socketConnected(socket *sockets.Socket, _ interface{}) bool {
	logger.Info("socket connected:%v", socket.Id())
	return true
}

func socketDisconnect(socket *sockets.Socket, _ interface{}) bool {
	logger.Info("socket disconnect:%v", socket.Id())
	return true
}

func socketDestroyed(socket *sockets.Socket, _ interface{}) bool {
	logger.Info("socket destroyed:%v", socket.Id())
	address := app.Config.GetString("address")
	_, _ = server.Connect(address) //重连
	return true
}

func ping(c *handler.Context) interface{} {
	var v string
	if err := c.Unmarshal(&v); err != nil {
		c.Socket.Errorf(err)
	} else {
		logger.Info("收到回复:%v %v", c.Path(), v)
	}

	return nil
}

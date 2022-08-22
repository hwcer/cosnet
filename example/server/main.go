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
	app.Start(&module{ModuleDefault: app.ModuleDefault{Id: "server"}})
	app.WaitForSystemExit()
}

type module struct {
	app.ModuleDefault
}

func (m *module) Start() error {
	address := app.Config.GetString("address")
	server = cosnet.New(app.SCC.Context, nil)
	_, err := server.Listen(address)
	if err != nil {
		return err
	}
	handle = server.Handler.(*handler.Handler)
	err = handle.Register(ping)
	if err != nil {
		return err
	}
	server.On(sockets.EventTypeError, socketError)
	server.On(sockets.EventTypeHeartbeat, socketHeartbeat)
	server.On(sockets.EventTypeConnected, socketConnected)
	server.On(sockets.EventTypeDisconnect, socketDisconnect)
	server.On(sockets.EventTypeDestroyed, socketDestroyed)
	return nil
}

func socketError(socket *sockets.Socket, err interface{}) bool {
	logger.Error("socket error:%v", err)
	return false
}
func socketHeartbeat(socket *sockets.Socket, _ interface{}) bool {
	socket.KeepAlive()
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
	return true
}

func ping(socket *sockets.Socket, msg *handler.Message) interface{} {
	var v string
	if err := msg.Unmarshal(&v); err != nil {
		socket.Errorf(err)
	} else {
		logger.Info("收到消息:%v %v", msg.Path(), v)
	}

	return "你好"
}

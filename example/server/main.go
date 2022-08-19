package main

import (
	"github.com/hwcer/cosgo/app"
	"github.com/hwcer/cosgo/logger"
	"github.com/hwcer/cosnet"
	"github.com/hwcer/cosnet/sockets"
)

var server *cosnet.Cosnet

func main() {
	app.Start(&module{ModuleDefault: app.ModuleDefault{Id: "server"}})
	app.WaitForSystemExit()
}

type module struct {
	app.ModuleDefault
}

func (m *module) Start() error {
	server = cosnet.New(app.SCC.Context, nil)
	_, err := server.Listen("tcp://0.0.0.0:3000")
	if err != nil {
		return err
	}
	err = server.Register(ping)
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

func ping(socket *sockets.Socket, msg sockets.Message) interface{} {
	var v string
	if err := msg.Unmarshal(&v); err != nil {
		socket.Errorf(err)
	} else {
		logger.Info("收到消息:%v %v", msg.Code(), v)
	}

	return "你好"
}

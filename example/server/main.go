package main

import (
	"github.com/hwcer/cosgo/app"
	"github.com/hwcer/cosgo/logger"
	"github.com/hwcer/cosnet"
	"github.com/hwcer/cosnet/sockets"
)

var server *cosnet.Cosnet

func main() {
	server = cosnet.New(app.SCC.Context, nil)
	_, err := server.Listen("tcp://0.0.0.0:3000")
	if err != nil {
		logger.Error(err)
	}
	server.On(sockets.EventTypeError, socketError)
	server.On(sockets.EventTypeConnected, socketConnected)
	server.On(sockets.EventTypeDisconnect, socketDisconnect)
	server.On(sockets.EventTypeDestroyed, socketDestroyed)
	app.WaitForSystemExit()
}

func socketError(socket *sockets.Socket, err interface{}) bool {
	logger.Error("socket error:%v", err)
	return false
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

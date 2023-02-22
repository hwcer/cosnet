package main

import (
	"github.com/hwcer/cosgo"
	"github.com/hwcer/cosgo/logger"
	"github.com/hwcer/cosnet"
	"github.com/spf13/pflag"
	"time"
)

var server *cosnet.Sockets

func init() {
	pflag.String("address", "tcp://0.0.0.0:3000", "server address")
}
func main() {
	cosgo.Start(&module{Module: cosgo.NewModule("server")})
	cosgo.WaitForSystemExit()
}

type module struct {
	*cosgo.Module
}

func (m *module) Start() error {
	address := cosgo.Config.GetString("address")
	server = cosnet.New(cosgo.SCC.Context)
	_, err := server.Listen(address)
	if err != nil {
		return err
	}
	err = server.Register(ping)
	if err != nil {
		return err
	}
	server.On(cosnet.EventTypeError, socketError)
	server.On(cosnet.EventTypeHeartbeat, socketHeartbeat)
	server.On(cosnet.EventTypeConnected, socketConnected)
	server.On(cosnet.EventTypeDisconnect, socketDisconnect)
	server.On(cosnet.EventTypeDestroyed, socketDestroyed)
	return nil
}

func socketError(socket *cosnet.Socket, err interface{}) bool {
	logger.Error("socket error:%v", err)
	return false
}
func socketHeartbeat(socket *cosnet.Socket, _ interface{}) bool {
	socket.KeepAlive()
	return true
}
func socketConnected(socket *cosnet.Socket, _ interface{}) bool {
	logger.Info("socket connected:%v", socket.Id())
	return true
}

func socketDisconnect(socket *cosnet.Socket, _ interface{}) bool {
	logger.Info("socket disconnect:%v", socket.Id())
	return true
}

func socketDestroyed(socket *cosnet.Socket, _ interface{}) bool {
	logger.Info("socket destroyed:%v", socket.Id())
	return true
}

func ping(c *cosnet.Context) interface{} {
	var v string
	if err := c.Bind(&v); err != nil {
		c.Socket.Errorf(err)
	} else {
		logger.Info("收到消息:%v %v", c.Path(), v)
	}

	return time.Now().Unix()
}

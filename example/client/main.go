package main

import (
	"context"
	"github.com/hwcer/cosgo"
	"github.com/hwcer/cosgo/scc"
	"github.com/hwcer/cosnet"
	"github.com/hwcer/cosnet/message"
	"github.com/hwcer/logger"
	"time"
)

var server *cosnet.Server

const C2SPing = "ping"

func init() {
	cosgo.Config.Flags("address", "", "tcp://127.0.0.1:3000", "server address")
	cosgo.Config.Flags("robot", "", 10, "server address")
}

func main() {
	cosgo.Start(true, &module{Module: cosgo.NewModule("client")})
}

type module struct {
	*cosgo.Module
}

var address string

func (m *module) Start() error {
	address = cosgo.Config.GetString("address")
	server = cosnet.New()
	//_ = server.Register(ping, C2SPing)
	server.On(cosnet.EventTypeError, socketError)
	server.On(cosnet.EventTypeMessage, socketMessage)
	server.On(cosnet.EventTypeHeartbeat, socketHeartbeat)
	server.On(cosnet.EventTypeConnected, socketConnected)
	server.On(cosnet.EventTypeDisconnect, socketDisconnect)
	server.On(cosnet.EventTypeDestroyed, socketDestroyed)

	robot := cosgo.Config.GetInt("robot")
	for i := 0; i < robot; i++ {
		scc.CGO(createRobot)
	}

	return nil
}

func createRobot(ctx context.Context) {
	sock, err := server.Connect(address)
	if err != nil {
		logger.Alert("Create Robot :%v", err)
		return
	}
	timer := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if err = sock.Send(C2SPing, "ping"); err != nil {
				logger.Alert("Robot Send:%v", err)
				return
			}
		}
	}
}

func socketMessage(socket *cosnet.Socket, i any) bool {
	msg := i.(message.Message)
	logger.Trace("收到未注册消息  PATH:%v   BODY:%v", msg.Path(), string(msg.Body()))
	return true
}
func socketError(socket *cosnet.Socket, err interface{}) bool {
	logger.Error("socket error:%v", err)
	return true
}

func socketHeartbeat(socket *cosnet.Socket, _ interface{}) bool {
	socket.KeepAlive()
	return true
}

func socketConnected(socket *cosnet.Socket, _ interface{}) bool {
	logger.Trace("socket connected:%v", socket.Id())
	return true
}

func socketDisconnect(socket *cosnet.Socket, _ interface{}) bool {
	logger.Trace("socket disconnect:%v", socket.Id())
	return true
}

func socketDestroyed(socket *cosnet.Socket, _ interface{}) bool {
	logger.Trace("socket destroyed:%v", socket.Id())
	_, _ = server.Connect(address) //重连
	return true
}

//func ping(c *cosnet.Context) interface{} {
//	//var v string
//	//if err := c.Unmarshal(&v); err != nil {
//	//	c.Socket.Errorf(err)
//	//} else {
//	//	logger.Info("收到回复:%v %v", c.Path(), v)
//	//}
//	//logger.Trace("收到回复  PATH:%v   BODY:%v", c.Message.Path(), string(c.Message.Body()))
//	return nil
//}

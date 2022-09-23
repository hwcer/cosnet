package main

import (
	"github.com/hwcer/cosgo"
	"github.com/hwcer/cosnet"
	"github.com/hwcer/logger"
	"github.com/spf13/pflag"
)

var server *cosnet.Agents

const C2SPing = "C2sHeartbeat"

func init() {
	pflag.String("address", "tcp://10.26.17.20:3001", "server address")
}

func main() {
	cosgo.Start(&module{Module: cosgo.NewModule("client")})
	cosgo.WaitForSystemExit()
}

type module struct {
	*cosgo.Module
}

func (m *module) Start() error {
	address := cosgo.Config.GetString("address")
	server = cosnet.New(cosgo.SCC.Context)
	_, err := server.Connect(address)
	if err != nil {
		return err
	}
	_ = server.Register(ping, C2SPing)
	server.On(cosnet.EventTypeError, socketError)
	server.On(cosnet.EventTypeHeartbeat, socketHeartbeat)
	server.On(cosnet.EventTypeConnected, socketConnected)
	server.On(cosnet.EventTypeDisconnect, socketDisconnect)
	server.On(cosnet.EventTypeDestroyed, socketDestroyed)
	return nil
}

func socketError(socket *cosnet.Socket, err interface{}) bool {
	logger.Error("socket error:%v", err)
	return true
}
func socketHeartbeat(socket *cosnet.Socket, _ interface{}) bool {
	socket.KeepAlive()
	m := socket.Agents.Acquire()
	if err := m.Marshal(0, C2SPing, "hi"); err == nil {
		socket.Write(m)
	}
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
	address := cosgo.Config.GetString("address")
	_, _ = server.Connect(address) //重连
	return true
}

func ping(c *cosnet.Context) interface{} {
	//var v string
	//if err := c.Unmarshal(&v); err != nil {
	//	c.Socket.Errorf(err)
	//} else {
	//	logger.Info("收到回复:%v %v", c.Path(), v)
	//}
	logger.Info("收到回复  PATH:%v   BODY:%v", []byte(c.Path()), c.Body())
	return nil
}

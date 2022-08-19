package sockets

import "encoding/json"

var Options = struct {
	MaxDataSize          uint32 //包体最大长度
	WriteChanSize        int32  //写通道缓存
	ConnectMaxSize       int32  //连接人数
	SocketHeartbeat      uint16 //(MS)服务器心跳,用来检测玩家僵尸连接
	SocketConnectTime    uint16 //(MS)没有动作被判断掉线
	SocketReconnectTime  uint16 //(MS)掉线后等待断线重连
	SocketDestroyingTime uint16 //(MS)强制踢掉线后登录最后消息发送完的时间
	//AuthenticationTime  uint16 //(MS)连接后等待登录的时间，默认 0：不需要登录
	//ReWriteTime         int32  //(MS)写消息失败时，重试间隔时间
	AutoCompressSize    uint32 //自动压缩
	ClientReconnectMax  uint16 //断线重连最大尝试次数
	ClientReconnectTime uint16 //断线重连每次等待时间(MS) ClientReconnectTime * ReconnectNum

	MessageMarshal   func(i interface{}) ([]byte, error) //默认消息解码
	MessageUnmarshal func(b []byte, i interface{}) error //默认消息编码
}{
	MaxDataSize:    1024 * 1024,
	WriteChanSize:  500,
	ConnectMaxSize: 50000,

	SocketHeartbeat:      2000,
	SocketConnectTime:    5000,
	SocketReconnectTime:  10000,
	SocketDestroyingTime: 1000,

	ClientReconnectMax:  1000,
	ClientReconnectTime: 5000,

	MessageMarshal:   json.Marshal,
	MessageUnmarshal: json.Unmarshal,
}

package cosnet

var Options = struct {
	MaxDataSize         int32  //包体最大长度
	WriteChanSize       int32  //写通道缓存
	ConnectMaxSize      int32  //连接人数
	SocketHeartbeat     uint16 //(MS)服务器心跳,用来检测玩家僵尸连接
	SocketConnectTime   uint16 //(MS)连接超时几次心跳没有动作被判断掉线
	SocketReconnectTime uint16 //(MS)掉线后等待断线重连
	//AuthenticationTime  uint16 //(MS)连接后等待登录的时间，默认 0：不需要登录
	//ReWriteTime         int32  //(MS)写消息失败时，重试间隔时间
	AutoCompressSize    uint32 //自动压缩
	ClientReconnectMax  uint16 //断线重连最大尝试次数
	ClientReconnectTime uint16 //断线重连每次等待时间(s) ClientReconnectTime * ReconnectNum
}{
	MaxDataSize:         1024 * 1024,
	WriteChanSize:       500,
	ConnectMaxSize:      50000,
	ClientReconnectMax:  1000,
	ClientReconnectTime: 5,
	SocketHeartbeat:     1000,
	SocketConnectTime:   10,
}

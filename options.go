package cosnet

var Options = struct {
	//AuthenticationTime  uint16 //(MS)连接后等待登录的时间，默认 0：不需要登录
	//ReWriteTime         int32  //(MS)写消息失败时，重试间隔时间
	WriteChanSize        int32  //写通道缓存
	ConnectMaxSize       int32  //连接人数
	SocketHeartbeat      uint32 //(MS)服务器心跳,用来检测玩家僵尸连接
	SocketConnectTime    uint32 //(MS)没有动作被判断掉线
	SocketDisconnectTime uint32 //(MS)掉线后等待断线重连
	AutoCompressSize     uint32 //自动压缩
	ClientReconnectMax   uint16 //断线重连最大尝试次数
	ClientReconnectTime  uint16 //断线重连每次等待时间(MS) ClientReconnectTime * ReconnectNum
	UdpServerWorker      int    //UDP工作进程数量
}{
	WriteChanSize:        100,
	ConnectMaxSize:       50000,
	SocketHeartbeat:      10000,
	SocketConnectTime:    5000,
	SocketDisconnectTime: 10000,
	ClientReconnectMax:   1000,
	ClientReconnectTime:  5000,
	UdpServerWorker:      64,
}

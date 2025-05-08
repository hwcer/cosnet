package cosnet

var S2CConfirm string //自动回包，确认包名，默认原路返回

var Options = struct {
	//AuthenticationTime  uint16 //(MS)连接后等待登录的时间，默认 0：不需要登录
	//ReWriteTime         int32  //(MS)写消息失败时，重试间隔时间
	WriteChanSize int32 //写通道缓存
	//ConnectMaxSize       int32  //连接人数
	SocketHeartbeat    int32 //(MS)服务器心跳,用来检测玩家僵尸连接
	SocketConnectTime  int32 //(MS)没有动作被判断掉线
	SocketReplacedTime int32 //(MS) 顶号延时关闭时间，
	//SocketDisconnectTime uint32 //(MS)掉线后等待断线重连
	AutoCompressSize    int32 //自动压缩
	ClientReconnectMax  int32 //断线重连最大尝试次数
	ClientReconnectTime int32 //断线重连每次等待时间(MS) ClientReconnectTime * ReconnectNum
	//UdpServerWorker      int    //UDP工作进程数量
}{
	WriteChanSize: 100,
	//ConnectMaxSize:       100000,
	SocketHeartbeat:    1000,
	SocketConnectTime:  60000,
	SocketReplacedTime: 5000,
	//SocketDisconnectTime: 10000,
	ClientReconnectMax:  1000,
	ClientReconnectTime: 5000,
	//UdpServerWorker:      64,
}

package cosnet

const RegistryMethod = "TCP"

var Options = struct {
	//AuthenticationTime  uint16 //(S)连接后等待登录的时间，默认 0：不需要登录
	//ReWriteTime         int32  //(S)写消息失败时，重试间隔时间
	Heartbeat           int32  //(S)服务器心跳,用来检测玩家僵尸连接
	S2CConfirm          string //确认包协议，默认原路返回(和请求时一致)
	WriteChanSize       int32  //写通道缓存
	ConnectMaxSize      int32  //连接人数
	SocketConnectTime   int32  //(S)没有动作被判断掉线
	SocketReplacedTime  int32  //(S) 顶号延时关闭时间，
	AutoCompressSize    int32  //自动压缩
	ClientReconnectMax  int32  //断线重连最大尝试次数
	ClientReconnectTime int32  //断线重连每次等待时间(MS) ClientReconnectTime * ReconnectNum
	//UdpServerWorker      int    //UDP工作进程数量
}{
	Heartbeat:          2,
	WriteChanSize:      100,
	ConnectMaxSize:     100000,
	SocketConnectTime:  30,
	SocketReplacedTime: 5,
	//SocketDisconnectTime: 10000,
	ClientReconnectMax:  1000,
	ClientReconnectTime: 5000,
	//UdpServerWorker:      64,
}

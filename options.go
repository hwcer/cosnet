package cosnet

// RegistryMethod 注册方法名称，默认为"TCP"
const RegistryMethod = "TCP"

// Options 配置选项结构体
var Options = struct {
	// Heartbeat 服务器心跳间隔，单位秒，用来检测玩家僵尸连接
	Heartbeat int32
	// WriteChanSize 写通道缓存大小
	WriteChanSize int32
	// ConnectMaxSize 最大连接人数
	ConnectMaxSize int32
	// SocketConnectTime 没有动作被判断为掉线的时间，单位秒
	SocketConnectTime int32
	// SocketReplacedTime 顶号延时关闭时间，单位秒
	SocketReplacedTime int32
	// AutoCompressSize 自动压缩的阈值，超过此大小的消息会被自动压缩
	AutoCompressSize int32
	// ClientReconnectMax 断线重连最大尝试次数
	ClientReconnectMax int32
	// ClientReconnectTime 断线重连每次等待时间，单位毫秒，实际等待时间为 ClientReconnectTime * 重连次数
	ClientReconnectTime int32
}{
	Heartbeat:           2,      // 心跳间隔 2 秒
	WriteChanSize:       100,    // 写通道缓存 100 条消息
	ConnectMaxSize:      100000, // 最大连接数 10 万
	SocketConnectTime:   30,     // 30 秒无动作判断为掉线
	SocketReplacedTime:  5,      // 顶号后 5 秒关闭旧连接
	ClientReconnectMax:  1000,   // 最大重连尝试 1000 次
	ClientReconnectTime: 5000,   // 每次重连等待 5 秒
}

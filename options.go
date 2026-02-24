package cosnet

// RegistryMethod 注册方法名称，默认为"TCP"
const RegistryMethod = "TCP"

type Config struct {
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

	// ClientReconnectMax 断线重连最大尝试次数，0 表示无限尝试
	ClientReconnectMax int32
	// ClientReconnectTime 断线重连基础等待时间，单位毫秒，实际等待时间为 ClientReconnectTime * 重连次数
	ClientReconnectTime int32
	// ClientReconnectMaxDelay 断线重连指数退避的最大等待时间，单位毫秒
	ClientReconnectMaxDelay int32
}

// Options 配置选项结构体
var Options = Config{
	Heartbeat:               10,     // 心跳间隔 10 秒
	WriteChanSize:           100,    // 写通道缓存 100 条消息
	ConnectMaxSize:          100000, // 最大连接数 10 万
	SocketConnectTime:       30,     // 30 秒无动作判断为掉线
	SocketReplacedTime:      5,      // 顶号后 5 秒关闭旧连接
	ClientReconnectMax:      10,     // 最大重连尝试 10 次
	ClientReconnectTime:     1000,   // 基础重连等待 1 秒（指数退避）
	ClientReconnectMaxDelay: 30000,  // 最大等待时间 30 秒
}

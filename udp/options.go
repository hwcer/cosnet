package udp

// Options UDP模块配置选项
var Options = struct {
	// ConnChanSize 连接通道缓存大小
	ConnChanSize int32
	// MsgChanSize 消息通道缓存大小
	MsgChanSize int32
}{
	ConnChanSize: 100, // 连接通道缓存 100 条消息
	MsgChanSize:  100, // 消息通道缓存 100 条消息
}

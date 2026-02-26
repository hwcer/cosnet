package message

type Flag uint8

// 标志位定义
const (
	FlagACK        Flag = 1 << iota // 需要确认
	FlagConfirm                     // 这是确认包
	FlagHeartbeat                   // 心跳包
	FlagBroadcast                   // 广播包
	FlagCompressed                  // 是否压缩
	FlagEncrypted                   // 是否加密
	FlagFragmented                  // 分片包
)

func (f Flag) Has(t Flag) bool {
	return f&t > 0
}

// Set bit位设置为1
func (f *Flag) Set(t Flag) {
	*f |= t
}

// Delete bit位设置为0
func (f *Flag) Delete(t Flag) {
	*f -= t
}

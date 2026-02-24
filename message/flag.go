package message

type Flag uint8

// 标志位定义
const (
	FlagCompressed  Flag = 1 << iota // 是否压缩
	FlagEncrypted                    // 是否加密
	FlagNeedACK                      // 需要确认
	FlagIsACK                        // 这是确认包
	FlagIsHeartbeat                  // 心跳包
	FlagIsBroadcast                  // 广播包
	FlagIsFragment                   // 分片包
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

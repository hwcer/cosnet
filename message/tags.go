package message

import (
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/logger"
)

type Tags uint8

const (
	TagsUseUrlPath     Tags = 1      //使用 path 模式,区别于协议码
	TagsServerMessage  Tags = 1 << 1 //由服务器推送的包,区别于客户端发送的包
	TagsEnableCompress Tags = 1 << 2 //启用压缩
	TagsEnableUnknown  Tags = 1 << 3
)

func (t *Tags) Has(i Tags) bool {
	return *t&i == i
}

func (t *Tags) Set(i Tags) {
	*t = *t | i
}
func (t *Tags) Del(i Tags) {
	s := i ^ 0xff
	*t = *t & s
}

func (t *Tags) Binder() binder.Binder {
	k := uint8(*t) & 0xf0
	if v, ok := Serialize[SerializeType(k)]; ok {
		return v
	}
	return Binder
}

func (t *Tags) SetBinder(k uint8) {
	if _, ok := Serialize[SerializeType(k)]; !ok {
		logger.Alert("unknown serialize key:%v", SerializeTypeJson)
		return
	}
	s := *t & 0xf
	*t = Tags(k & uint8(s))
}

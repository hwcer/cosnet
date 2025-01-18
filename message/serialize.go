package message

import (
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/logger"
)

type SerializeType uint8

const (
	SerializeTypeJson     SerializeType = 1 << 4
	SerializeTypeProtobuf SerializeType = 1<<4 + 1
)

var Serialize = serialize{}

type serialize map[SerializeType]binder.Binder

func init() {
	Serialize.Register(SerializeTypeJson, binder.Json)
	Serialize.Register(SerializeTypeProtobuf, binder.Protobuf)
}

func (s serialize) Register(k SerializeType, b binder.Binder) {
	if k < SerializeTypeJson {
		logger.Alert("Serialize Register minimum value of key is %v", SerializeTypeJson)
		return
	}
	if _, ok := s[k]; ok {
		logger.Alert("Serialize Register key exist %v", k)
		return
	}
	s[k] = b
}

package message

import (
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/logger"
)

const (
	MagicNumberPathJson byte = 0x80
	MagicNumberCodeJson byte = 0x81
)

type MagicType int8

const (
	MagicTypePath MagicType = 0
	MagicTypeCode MagicType = 1
)

var Magics = magics{}

func init() {
	Magics.Register(MagicNumberPathJson, MagicTypePath, binder.Json)
	Magics.Register(MagicNumberCodeJson, MagicTypeCode, binder.Json)
}

type Magic struct {
	Key    byte
	Type   MagicType     //工作模式:0-path, 1-code
	Binder binder.Binder //序列化方式
}

type magics map[byte]*Magic

func (ms magics) Has(key byte) bool {
	_, ok := ms[key]
	return ok
}
func (ms magics) Get(key byte) *Magic {
	return ms[key]
}

func (ms magics) Register(key byte, mode MagicType, b binder.Binder) {
	if _, ok := ms[key]; ok {
		logger.Alert("Magic Number exists:%s", string(key))
		return
	}
	m := new(Magic)
	m.Key = key
	m.Type = mode
	m.Binder = b
	ms[key] = m
}

package message

import (
	"encoding/binary"
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
	Magics.Register(MagicNumberPathJson, MagicTypePath, binder.Json, binary.BigEndian)
	Magics.Register(MagicNumberCodeJson, MagicTypeCode, binder.Json, binary.BigEndian)
}

type Magic struct {
	Key    byte
	Type   MagicType        //工作模式:0-path, 1-code
	Binder binder.Binder    //序列化方式
	Binary binary.ByteOrder //大端 or 小端
}

type magics map[byte]*Magic

func (ms magics) Has(key byte) bool {
	_, ok := ms[key]
	return ok
}
func (ms magics) Get(key byte) *Magic {
	return ms[key]
}

func (ms magics) Register(key byte, mode MagicType, bi binder.Binder, by binary.ByteOrder) {
	if _, ok := ms[key]; ok {
		logger.Alert("Magic Number exists:%s", string(key))
		return
	}
	m := new(Magic)
	m.Key = key
	m.Type = mode
	m.Binder = bi
	m.Binary = by
	ms[key] = m
}

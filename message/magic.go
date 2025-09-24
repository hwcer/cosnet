package message

import (
	"encoding/binary"

	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/logger"
)

const (
	MagicNumberPathJsonConfirm byte = 0xf0
	MagicNumberPathJsonNoreply byte = 0xf1
	MagicNumberCodeJsonConfirm byte = 0xf2
	MagicNumberCodeJsonNoreply byte = 0xf3
)

type MagicType int8

const (
	MagicTypePath MagicType = 0
	MagicTypeCode MagicType = 1
)

var Magics = magics{}

func init() {
	Magics.Register(MagicNumberPathJsonConfirm, MagicTypePath, true, binder.Json, binary.BigEndian)
	Magics.Register(MagicNumberPathJsonNoreply, MagicTypePath, false, binder.Json, binary.BigEndian)

	Magics.Register(MagicNumberCodeJsonConfirm, MagicTypeCode, true, binder.Json, binary.BigEndian)
	Magics.Register(MagicNumberCodeJsonNoreply, MagicTypeCode, false, binder.Json, binary.BigEndian)
}

type Magic struct {
	Key     byte
	Type    MagicType        //工作模式:0-path, 1-code
	Binder  binder.Binder    //序列化方式
	Binary  binary.ByteOrder //大端 or 小端
	Confirm bool             //是否需要确认包
}

type magics map[byte]*Magic

func (ms magics) Has(key byte) bool {
	_, ok := ms[key]
	return ok
}
func (ms magics) Get(key byte) *Magic {
	return ms[key]
}

func (ms magics) Register(key byte, mt MagicType, confirm bool, bi binder.Binder, by binary.ByteOrder) {
	if _, ok := ms[key]; ok {
		logger.Alert("Magic Number exists:%s", string(key))
		return
	}
	m := new(Magic)
	m.Key = key
	m.Type = mt
	m.Binder = bi
	m.Binary = by
	m.Confirm = confirm
	ms[key] = m
}

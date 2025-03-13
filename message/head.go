package message

import (
	"encoding/binary"
	"fmt"
	"github.com/hwcer/cosgo/binder"
)

const messageHeadSize = 12

const (
	ModeConfirm  uint8 = 1
	ModeCompress uint8 = 1 << 1
)

type Head struct {
	magic byte   //1
	mode  uint8  //1需要回复
	code  uint16 //2 协议码、PATH 长度
	size  uint32 //4 BODY总长度(包含PATH)
	index uint32 //4 client_id,server_id
}

// Size 包体总长
func (h *Head) Size() uint32 {
	return h.size
}

func (h *Head) Index() uint32 {
	return h.index
}
func (h *Head) Magic() *Magic {
	return Magics[h.magic]
}

// Confirm 是否需要回复
func (h *Head) Confirm(v ...bool) bool {
	if len(v) > 0 {
		if v[0] {
			h.mode = h.mode | ModeConfirm
		} else {
			h.mode = h.mode - 1
		}
	}
	return h.mode&ModeConfirm == ModeConfirm
}

// Parse 解析二进制头并填充到对应字段
func (h *Head) Parse(head []byte) error {
	if len(head) != messageHeadSize {
		return ErrMsgHeadIllegal
	}
	if !Magics.Has(head[0]) {
		return ErrMsgHeadIllegal
	}
	h.magic = head[0]
	h.mode = uint8(head[1])
	h.code = binary.BigEndian.Uint16(head[2:4])
	h.size = binary.BigEndian.Uint32(head[4:8])
	h.index = binary.BigEndian.Uint32(head[8:12])
	if h.size > Options.MaxDataSize {
		return ErrMsgDataSizeTooLong
	}
	return nil
}
func (h *Head) bytes() []byte {
	head := make([]byte, messageHeadSize)
	head[0] = h.magic
	head[1] = byte(h.mode)
	binary.BigEndian.PutUint16(head[2:4], h.code)
	binary.BigEndian.PutUint32(head[4:8], h.size)
	binary.BigEndian.PutUint32(head[8:12], h.index)
	return head
}

func (h *Head) format(magic byte, index uint32, path string) (err error) {
	h.magic = magic
	h.index = index

	mc := Magics.Get(h.magic)
	if mc == nil {
		return fmt.Errorf("message magic not exist,Path:%s,Magic:%d", path, uint8(h.magic))
	}

	if mc.Type == MagicTypePath {
		h.code = uint16(len(path))
	} else {
		h.code, err = Transform.Code(path)
	}
	return
}
func (h *Head) Binder() binder.Binder {
	magic := h.Magic()
	if magic == nil {
		return nil
	}
	return magic.Binder
}

func (h *Head) Release() {
	h.mode = 0
	h.code = 0
	h.size = 0
	h.index = 0
}

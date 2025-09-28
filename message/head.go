package message

import (
	"fmt"

	"github.com/hwcer/cosgo/binder"
)

const messageHeadSize = 9

type Head struct {
	magic byte   //1
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

// Parse 解析二进制头并填充到对应字段
func (h *Head) Parse(head []byte) error {
	if len(head) != messageHeadSize {
		return ErrMsgHeadIllegal
	}
	magic := Magics.Get(head[0])
	if magic == nil {
		return ErrMsgHeadIllegal
	}
	h.magic = head[0]
	h.size = magic.Binary.Uint32(head[1:5])
	h.index = magic.Binary.Uint32(head[5:9])
	if h.size > Options.MaxDataSize {
		return ErrMsgDataSizeTooLong
	}
	return nil
}
func (h *Head) bytes() []byte {
	magic := h.Magic()
	head := make([]byte, messageHeadSize)
	head[0] = h.magic
	magic.Binary.PutUint32(head[1:5], h.size)
	magic.Binary.PutUint32(head[5:9], h.index)
	return head
}

func (h *Head) format(magic byte, index uint32) (err error) {
	h.magic = magic
	h.index = index

	mc := Magics.Get(h.magic)
	if mc == nil {
		return fmt.Errorf("message magic not exist,Magic:%d", h.magic)
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
	h.size = 0
	h.index = 0
}

package message

import (
	"encoding/binary"
	"strconv"
)

const messageHeadSize = 12

const (
	HeadTag   = "_msg_h_tag"
	HeadIndex = "_msg_h_idx"
)

type Head struct {
	tags  Tags   //1
	code  uint16 //2 协议码、PATH 长度
	size  uint32 // 4 BODY总长度(包含PATH)
	index uint32 // 4 client_id,server_id
}

// Size 包体总长
func (h *Head) Size() uint32 {
	return h.size
}
func (h *Head) Query() map[string]string {
	r := make(map[string]string)
	r[HeadTag] = strconv.FormatUint(uint64(h.tags), 10)
	r[HeadIndex] = strconv.FormatUint(uint64(h.index), 10)
	return r
}

// Parse 解析二进制头并填充到对应字段
func (h *Head) Parse(head []byte) error {
	if len(head) != messageHeadSize {
		return ErrMsgHeadIllegal
	}
	if head[0] != MagicNumber {
		return ErrMsgHeadIllegal
	}
	h.tags = Tags(head[1])
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
	head[0] = MagicNumber
	head[1] = byte(h.tags)
	binary.BigEndian.PutUint16(head[2:4], h.code)
	binary.BigEndian.PutUint32(head[4:8], h.size)
	binary.BigEndian.PutUint32(head[8:12], h.index)
	return head
}

func (h *Head) format(path string, query map[string]string) (err error) {
	if i, ok := query[HeadTag]; ok {
		h.tags = Tags(i[0])
	}
	if i, ok := query[HeadIndex]; ok {
		n, _ := strconv.Atoi(i)
		h.index = uint32(n)
	}

	if h.tags.Has(TagsUseUrlPath) {
		h.code = uint16(len(path))
	} else {
		h.code, err = Transform.Code(path)
	}
	return
}

func (h *Head) Release() {
	h.tags = 0
	h.code = 0
	h.size = 0
	h.index = 0
}

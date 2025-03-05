package message

import (
	"encoding/binary"
	"github.com/hwcer/cosgo/binder"
	"strconv"
)

const messageHeadSize = 12

const (
	HeadAtomicIndex = "_m_h_idx"
	HeadMessageCode = "_m_h_code"
)

type Head struct {
	magic byte   //1
	typ   uint8  //编码方式
	code  uint16 //2 协议码、PATH 长度
	size  uint32 //4 BODY总长度(包含PATH)
	index uint32 //4 client_id,server_id
}

// Size 包体总长
func (h *Head) Size() uint32 {
	return h.size
}
func (h *Head) Query() map[string]string {
	r := make(map[string]string)
	if t := binder.Type(h.typ); t != nil {
		r[binder.HeaderContentType] = t.Name
	}
	if Options.Mode == HeadModeCode {
		r[HeadMessageCode] = strconv.Itoa(int(h.code))
	}
	r[HeadAtomicIndex] = strconv.FormatUint(uint64(h.index), 10)
	return r
}

// Confirm 是否需要回复
func (h *Head) Confirm() bool {
	return h.magic == MagicConfirm
}

// Parse 解析二进制头并填充到对应字段
func (h *Head) Parse(head []byte) error {
	if len(head) != messageHeadSize {
		return ErrMsgHeadIllegal
	}
	if head[0] != MagicNumber && head[0] != MagicConfirm {
		return ErrMsgHeadIllegal
	}
	h.magic = head[0]
	h.typ = uint8(head[1])
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
	head[1] = byte(h.typ)
	binary.BigEndian.PutUint16(head[2:4], h.code)
	binary.BigEndian.PutUint32(head[4:8], h.size)
	binary.BigEndian.PutUint32(head[8:12], h.index)
	return head
}

func (h *Head) format(path string, meta map[string]string) (err error) {
	if i, ok := meta[binder.HeaderContentType]; ok {
		if t := binder.Type(i); t != nil {
			h.typ = t.Id
		}
	}

	if i, ok := meta[HeadAtomicIndex]; ok {
		n, _ := strconv.Atoi(i)
		h.index = uint32(n)
	}

	if Options.Mode == HeadModePath {
		h.code = uint16(len(path))
	} else {
		h.code, err = Transform.Code(path)
	}
	return
}

func (h *Head) Release() {
	h.typ = 0
	h.code = 0
	h.size = 0
	h.index = 0
}

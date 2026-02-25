package message

import (
	"fmt"

	"github.com/hwcer/cosgo/binder"
)

const messageHeadSize = 10

type Head struct {
	magic byte  //1
	flag  Flag  //1
	size  int32 //4 BODY总长度(包含PATH) || code
	index int32 //4 client_id,server_id
}

func (h *Head) Flag() Flag {
	return h.flag
}

// Size 包体总长
func (h *Head) Size() int32 {
	return h.size
}

func (h *Head) Index() int32 {
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
	h.flag = Flag(head[1])                           // 解析 tags 字段
	h.size = int32(magic.Binary.Uint32(head[2:6]))   // 调整 size 字段位置
	h.index = int32(magic.Binary.Uint32(head[6:10])) // 调整 index 字段位置
	if h.size > Options.MaxDataSize {
		return ErrMsgDataSizeTooLong
	}
	return nil
}

// bytes 生成二进制头，返回头部数据和是否启用压缩
func (h *Head) bytes() ([]byte, bool) {
	magic := h.Magic()
	head := make([]byte, messageHeadSize)
	head[0] = h.magic
	flag := h.flag

	// 检查是否需要添加压缩标记
	compressed := false
	if Options.AutoCompressSize > 0 && h.size > Options.AutoCompressSize && !flag.Has(FlagCompressed) {
		flag.Set(FlagCompressed)
		compressed = true
	}

	head[1] = uint8(flag)                               // 写入 tags 字段
	magic.Binary.PutUint32(head[2:6], uint32(h.size))   // 调整 size 字段位置
	magic.Binary.PutUint32(head[6:10], uint32(h.index)) // 调整 index 字段位置
	return head, compressed
}

func (h *Head) format(magic byte, flag Flag, index int32) (err error) {
	h.magic = magic
	h.flag = flag
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
	h.magic = 0 // 重置 magic 字段
	h.flag = 0  // 重置 tags 字段
	h.size = 0
	h.index = 0
}

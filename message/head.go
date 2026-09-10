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
	//size为uint32转int32,大值会变负数;负数或超长均视为非法包,否则后续按size切片会panic
	if h.size < 0 || h.size > Options.MaxDataSize {
		return ErrMsgDataSizeTooLong
	}
	return nil
}

// bytes 生成二进制头
// wireSize 为线上实际传输的包体长度(启用压缩时为压缩后长度):
// 接收端TCP按此长度ReadFull,若与实际字节数不一致会导致流错位
// compressed 表示线上数据体是否为gzip压缩,须与实际写入的数据保持一致
func (h *Head) bytes(wireSize int32, compressed bool) []byte {
	magic := h.Magic()
	head := make([]byte, messageHeadSize)
	head[0] = h.magic
	flag := h.flag

	// 检查是否需要添加压缩标记
	if compressed {
		flag.Set(FlagCompressed)
	}

	head[1] = uint8(flag)                               // 写入 tags 字段
	magic.Binary.PutUint32(head[2:6], uint32(wireSize)) // 调整 size 字段位置
	magic.Binary.PutUint32(head[6:10], uint32(h.index)) // 调整 index 字段位置
	return head
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

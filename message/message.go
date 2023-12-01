package message

import (
	"bytes"
	"encoding/binary"
	"io"
	"sync"
)

const (
	HeadSize = 7
)

func New() Handler {
	i := &handler{pool: sync.Pool{}}
	i.pool.New = i.New
	return i
}

type handler struct {
	pool sync.Pool
}

func (h *handler) New() any {
	return &message{}
}

func (h *handler) Head() []byte {
	return make([]byte, HeadSize)
}
func (h *handler) Require() Message {
	i := h.pool.Get()
	return i.(Message)
}

func (h *handler) Release(i Message) {
	if v, ok := i.(*message); ok {
		v.path = 0
		v.body = 0
		h.pool.Put(v)
	}
}

// message 默认使用路径模式集中注册协议
// path : path路径字节数
// body : body数据字节数
// data : path + body
// path : /ping?t=1  URL路径模式，没有实际属性名，和body共同组成data

type message struct {
	path uint16 //2    路径PATH长度
	body uint32 //4    数据BODY长度
	data []byte //数据   [path][body]
}

// Len 包体总长
func (m *message) Len() int {
	return int(m.path) + int(m.body)
}

// Path 路径
func (m *message) Path() string {
	return string(m.data[0:m.path])
}

// Body 消息体数据  [0,,,,10,11,12 .....]
func (m *message) Body() []byte {
	return m.data[m.path:]
}

// Parse 解析二进制头并填充到对应字段
func (m *message) Parse(head []byte) error {
	if head[0] != Options.MagicNumber {
		return ErrMsgHeadIllegal
	}
	m.path = binary.BigEndian.Uint16(head[1:3])
	m.body = binary.BigEndian.Uint32(head[3:7])
	if m.body > Options.MaxDataSize {
		return ErrMsgDataSizeTooLong
	}
	return nil
}

// Bytes 生成二进制文件
func (m *message) Bytes(w io.Writer) (n int, err error) {
	size := m.Len()
	head := make([]byte, HeadSize)
	head[0] = Options.MagicNumber
	binary.BigEndian.PutUint16(head[1:3], m.path)
	binary.BigEndian.PutUint32(head[3:7], m.body)
	var r int
	if r, err = w.Write(head); err == nil {
		n += r
	} else {
		return
	}
	if size > 0 {
		r, err = w.Write(m.data[0:size])
		n += r
	}
	return
}

// Write 从conn中读取数据写入到data
func (m *message) Write(r io.Reader) (n int, err error) {
	size := m.Len()
	if size == 0 {
		return
	}
	if cap(m.data) >= size {
		m.data = m.data[0:size]
	} else if size > Options.Capacity {
		m.data = make([]byte, size)
	} else {
		m.data = make([]byte, size, Options.Capacity)
	}
	return io.ReadFull(r, m.data)
}

// Marshal 将一个对象放入Message.data
func (m *message) Marshal(path string, body any) error {
	buffer := bytes.NewBuffer(m.data[:0])
	if n, err := buffer.WriteString(path); err == nil {
		m.path = uint16(n)
	} else {
		return err
	}
	var err error
	switch v := body.(type) {
	case []byte:
		_, err = buffer.Write(v)
	default:
		err = Options.Binder.Encode(buffer, body)
	}
	if err != nil {
		return err
	}
	m.body = uint32(buffer.Len() - int(m.path))
	m.data = buffer.Bytes()
	return nil
}

// Unmarshal 解析Message body
func (m *message) Unmarshal(i interface{}) (err error) {
	return Options.Binder.Unmarshal(m.Body(), i)
}

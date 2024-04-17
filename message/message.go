package message

import (
	"bytes"
	"encoding/binary"
	"io"
)

const messageHeadSize = 5

// message 默认使用路径模式集中注册协议
// path : path路径字节数
// body : body数据字节数
// bytes : path + body

// path : /ping?t=1  URL路径模式，没有实际属性名，和body共同组成data

type message struct {
	size  int32  //4    bytes长度
	bytes []byte //数据   [int32][path][body]
	l     int    //path
}

// Size 包体总长
func (m *message) Size() int32 {
	return m.size
}

func (m *message) length() int {
	if m.l == 0 {
		m.l = int(binary.BigEndian.Uint32(m.bytes[0:4]))
	}
	return m.l
}

// Path 路径
func (m *message) Path() string {
	s := m.length() + 4
	return string(m.bytes[4:s])
}

// Body 消息体数据  [0,,,,10,11,12 .....]
func (m *message) Body() []byte {
	s := m.length() + 4
	return m.bytes[s:]
}

// Parse 解析二进制头并填充到对应字段
func (m *message) Parse(head []byte) error {
	if head[0] != Options.MagicNumber {
		return ErrMsgHeadIllegal
	}
	m.size = int32(binary.BigEndian.Uint32(head[1:5]))
	if m.size > Options.MaxDataSize {
		return ErrMsgDataSizeTooLong
	}
	return nil
}

// Bytes 生成二进制文件
func (m *message) Bytes(w io.Writer) (n int, err error) {
	size := m.Size()
	head := make([]byte, messageHeadSize)
	head[0] = Options.MagicNumber
	binary.BigEndian.PutUint32(head[1:5], uint32(m.size))
	var r int
	if r, err = w.Write(head); err == nil {
		n += r
	} else {
		return
	}
	if size > 0 {
		r, err = w.Write(m.bytes[0:size])
		n += r
	}
	return
}

// Write 从conn中读取数据写入到data
func (m *message) Write(r io.Reader) (n int, err error) {
	size := int(m.Size())
	if size == 0 {
		return
	}
	if cap(m.bytes) >= size {
		m.bytes = m.bytes[0:size]
	} else if size > Options.Capacity {
		m.bytes = make([]byte, size)
	} else {
		m.bytes = make([]byte, size, Options.Capacity)
	}
	return io.ReadFull(r, m.bytes)
}

// Marshal 将一个对象放入Message.data
func (m *message) Marshal(path string, body any) error {
	if len(m.bytes) < 4 {
		m.bytes = make([]byte, 0, Options.Capacity)
	}
	b := []byte(path)
	m.l = len(b)
	binary.BigEndian.PutUint32(m.bytes[0:4], uint32(m.l))
	buffer := bytes.NewBuffer(m.bytes[0:4])
	if _, err := buffer.Write(b); err != nil {
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

	m.size = int32(buffer.Len())
	m.bytes = buffer.Bytes()
	return nil
}
func (m *message) Reset(b []byte) {
	m.size = int32(len(b))
	m.bytes = b
}

// Unmarshal 解析Message body
func (m *message) Unmarshal(i interface{}) (err error) {
	return Options.Binder.Unmarshal(m.Body(), i)
}

func (m *message) Release() {
	m.size = 0
	m.l = 0
}

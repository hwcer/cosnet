package message

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const messageHeadSize = 5

const messagePathSize = 2  //Path len
const messageIndexSize = 4 //index len

// message 默认使用路径模式集中注册协议

// head  [MagicNumber,uint32(bytes)]
// body : body数据字节数
// bytes :  len(path) + path + body

// path : /ping?t=1  URL路径模式，没有实际属性名，和body共同组成data

// message 默认使用路径模式集中注册协议
// path : path路径字节数
// body : body数据字节数
// bytes : path + body

// path : /ping?t=1  URL路径模式，没有实际属性名，和body共同组成data

type message struct {
	size  uint32 //4    bytes长度
	bytes []byte //数据   [int8][path][body]
	l     []int  //索引  [path开始位置,path结束位置不包括]
}

// Size 包体总长
func (m *message) Size() uint32 {
	return m.size
}
func (m *message) Index() uint32 {
	return binary.BigEndian.Uint32(m.bytes[0:messageIndexSize])
}

func (m *message) length() []int {
	if m.l == nil {
		m.l = make([]int, 2)
		m.l[0] = messageIndexSize + messagePathSize
		m.l[1] = int(binary.BigEndian.Uint16(m.bytes[messageIndexSize:m.l[0]])) + m.l[0]
	}
	return m.l
}

// Path 路径
func (m *message) Path() string {
	s := m.length()
	return string(m.bytes[s[0]:s[1]])
}

// Body 消息体数据  [0,,,,10,11,12 .....]
func (m *message) Body() []byte {
	s := m.length()
	return m.bytes[s[1]:]
}

// Verify 校验包体是否正常
func (m *message) Verify() error {
	s := m.length()
	if l := len(m.bytes); l < s[1] {
		if l > 255 {
			l = 255
		}
		return fmt.Errorf("message too short:%v", string(m.bytes[0:l]))
	}
	return nil
}

// Parse 解析二进制头并填充到对应字段
func (m *message) Parse(head []byte) error {
	if len(head) != messageHeadSize {
		return ErrMsgHeadIllegal
	}
	if head[0] != Options.MagicNumber {
		return ErrMsgHeadIllegal
	}
	m.size = binary.BigEndian.Uint32(head[1:5])
	if m.size > Options.MaxDataSize {
		return ErrMsgDataSizeTooLong
	}
	return nil
}

// Bytes 生成二进制文件
func (m *message) Bytes(w io.Writer, includeHeader bool) (n int, err error) {
	var r int
	size := m.Size()
	if includeHeader {
		head := Options.Head()
		head[0] = Options.MagicNumber
		binary.BigEndian.PutUint32(head[1:5], m.size)
		if r, err = w.Write(head); err == nil {
			n += r
		} else {
			return
		}
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
	n, err = io.ReadFull(r, m.bytes[0:size])
	if n != size {
		return n, io.ErrShortBuffer
	}
	return
}

// Marshal 将一个对象放入Message.data
func (m *message) Marshal(index uint32, path string, body any) error {
	b := []byte(path)
	m.l = make([]int, 2)
	m.l[0] = messageIndexSize + messagePathSize
	m.l[1] = m.l[0] + len(b)

	head := make([]byte, m.l[0])

	binary.BigEndian.PutUint32(head[0:messageIndexSize], index)
	binary.BigEndian.PutUint16(head[messageIndexSize:m.l[0]], uint16(len(b)))

	buffer := bytes.NewBuffer(m.bytes[0:0])
	buffer.Write(head)
	buffer.Write(b)

	var err error
	switch v := body.(type) {
	case []byte:
		buffer.Write(v)
	default:
		err = Options.Binder.Encode(buffer, body)
	}
	if err != nil {
		return err
	}
	m.bytes = buffer.Bytes()
	m.size = uint32(len(m.bytes))
	return nil
}
func (m *message) Reset(b []byte) {
	m.size = uint32(len(b))
	m.bytes = b
}

// Unmarshal 解析Message body
func (m *message) Unmarshal(i interface{}) (err error) {
	return Options.Binder.Unmarshal(m.Body(), i)
}

func (m *message) Release() {
	m.size = 0
	m.l = nil
}

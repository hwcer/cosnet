package message

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const messageHeadSize = 5

// message 默认使用路径模式集中注册协议

// head  [MagicNumber,uint32(bytes)]
// body : body数据字节数
// bytes :  len(path) + path + body

// path : /ping?t=1  URL路径模式，没有实际属性名，和body共同组成data

type message struct {
	i     int    // path start index
	j     int    // body start index
	size  uint32 //4    bytes长度
	bytes []byte //数据   [int32][path][body]

}

// Size 包体总长
func (m *message) Size() uint32 {
	return m.size
}

func (m *message) parse() (i int, j int) {
	if m.i == 0 {
		if m.i = bytes.IndexByte(m.bytes, '/'); m.i >= 0 {
			var e error
			if m.j, e = strconv.Atoi(string(m.bytes[:m.i])); e == nil {
				m.j += m.i
			} else {
				m.i = -1
			}
		} else {
			m.i = -1
		}
	}
	return m.i, m.j
}

// Path 路径
func (m *message) Path() string {
	i, j := m.parse()
	if i <= 0 {
		return ""
	}
	return string(m.bytes[i:j])
}

// Body 消息体数据  [0,,,,10,11,12 .....]
func (m *message) Body() []byte {
	i, j := m.parse()
	if i <= 0 {
		return nil
	}
	return m.bytes[j:]
}

// Verify 校验包体是否正常
func (m *message) Verify() error {
	i, j := m.parse()
	if i <= 0 {
		return nil
	}
	if l := len(m.bytes); l < j {
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
		head := make([]byte, messageHeadSize)
		head[0] = Options.MagicNumber
		binary.BigEndian.PutUint32(head[1:5], uint32(m.size))
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
func (m *message) Marshal(path string, body any) error {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	b := []byte(path)
	l := len(b)
	s := strconv.Itoa(l)
	m.i = len(s)
	m.j = m.i + l
	buffer := bytes.NewBuffer(m.bytes)
	buffer.Reset()
	buffer.WriteString(s)
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
	m.i = 0
	m.j = 0
}

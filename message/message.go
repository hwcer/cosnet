package message

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/hwcer/cosgo/binder"
	"github.com/hwcer/cosgo/values"
	"io"
	"net/url"
	"strings"
)

const messageHeadSize = 5

const messagePathSize = 2 //Path len
//const messageIndexSize = 4 //index len

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
	l     uint16 //索引   path.len
	u     *url.URL
}

// Size 包体总长
func (m *message) Size() uint32 {
	return m.size
}

//func (m *message) Index() uint32 {
//	return binary.BigEndian.Uint32(m.bytes[0:messageIndexSize])
//}

func (m *message) length() int {
	if m.l == 0 {
		m.l = binary.BigEndian.Uint16(m.bytes[0:messagePathSize])
	}
	return int(m.l)
}

func (m *message) Url() *url.URL {
	if m.u != nil {
		return m.u
	}
	s := m.length() + messagePathSize
	path := string(m.bytes[messagePathSize:s])
	i, err := url.Parse(path)
	if err != nil {
		m.u = &url.URL{Path: "/", RawPath: "/"}
	} else {
		m.u = i
	}
	return m.u
}

// Path 路径
func (m *message) Path() (r string, err error) {
	return m.Url().RawPath, nil
}
func (m *message) Query() values.Values {
	r := make(values.Values)
	q := m.Url().Query()
	for k, _ := range q {
		r[k] = q.Get(k)
	}
	return r
}

// Body 消息体数据  [0,,,,10,11,12 .....]
func (m *message) Body() []byte {
	s := m.length() + messagePathSize
	return m.bytes[s:]
}

// Verify 校验包体是否正常
func (m *message) Verify() error {
	s := m.length() + messagePathSize
	if l := len(m.bytes); l < s {
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
func (m *message) Marshal(path string, query values.Values, body any, bs ...binder.Binder) error {
	if len(query) > 0 {
		b := strings.Builder{}
		b.WriteString(path)
		b.WriteString("?")
		q := url.Values{}
		for k, _ := range query {
			q.Set(k, query.GetString(k))
		}
		b.WriteString(q.Encode())
		path = b.String()
	}

	b := []byte(path)
	l := len(m.bytes)
	m.l = uint16(l)
	head := make([]byte, messagePathSize)
	binary.BigEndian.PutUint16(head[0:l], m.l)
	buffer := bytes.NewBuffer(m.bytes[0:0])
	buffer.Write(head)
	buffer.Write(b)

	var err error
	switch v := body.(type) {
	case []byte:
		buffer.Write(v)
	default:
		bi := m.Binder(bs...)
		err = bi.Encode(buffer, body)
	}
	if err != nil {
		return err
	}
	m.bytes = buffer.Bytes()
	m.size = uint32(len(m.bytes))
	return nil
}
func (m *message) Reset(b []byte) error {
	m.size = uint32(len(b))
	m.bytes = b
	return nil
}

// Unmarshal 解析Message body
func (m *message) Unmarshal(i any, bs ...binder.Binder) (err error) {
	bi := m.Binder(bs...)
	return bi.Unmarshal(m.Body(), i)
}

func (m *message) Release() {
	m.size = 0
	m.l = 0
	m.u = nil
}

func (m *message) Binder(bs ...binder.Binder) binder.Binder {
	if len(bs) > 0 && bs[0] != nil {
		return bs[0]
	}
	return Options.Binder
}

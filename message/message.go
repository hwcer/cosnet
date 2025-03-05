package message

import (
	"bytes"
	"fmt"
	"github.com/hwcer/cosgo/binder"
	"io"
)

//const messageHeadSize = 5

//const messagePathSize = 2 //Path len
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
	Head
	bytes []byte //数据
}

// Path 路径
func (m *message) Path() (r string, err error) {
	if Options.Mode == HeadModePath {
		r = string(m.bytes[0:int(m.code)])
	} else {
		r, err = Transform.Path(m.code)
	}
	return
}

// Body 消息体
func (m *message) Body() []byte {
	if Options.Mode == HeadModePath {
		return m.bytes[int(m.code):]
	} else {
		return m.bytes
	}
}

// Verify 校验包体是否正常
func (m *message) Verify() error {
	if l := len(m.bytes); l != int(m.size) {
		return fmt.Errorf("message too short:%v", string(m.bytes))
	}
	return nil
}

// Bytes 生成二进制文件
func (m *message) Bytes(w io.Writer, includeHeader bool) (n int, err error) {
	var r int
	size := m.Size()
	if includeHeader {
		head := m.Head.bytes()
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

// Reset  WS UDP 数据包模式直接填充
func (m *message) Reset(b []byte) error {
	if len(b) < messageHeadSize {
		return io.ErrUnexpectedEOF
	}
	if err := m.Head.Parse(b[0:messageHeadSize]); err != nil {
		return err
	}
	m.bytes = b[messageHeadSize:]
	return nil
}

// Marshal 将一个对象放入Message.data
func (m *message) Marshal(path string, body any, meta map[string]string) (err error) {
	if err = m.Head.format(path, meta); err != nil {
		return
	}
	buffer := bytes.NewBuffer(m.bytes[0:0])
	if Options.Mode == HeadModePath {
		buffer.WriteString(path)
	}
	switch v := body.(type) {
	case []byte:
		if len(v) > 0 {
			buffer.Write(v)
		}
	default:
		bi := binder.GetContentType(meta, binder.ContentTypeModReq)
		err = bi.Encode(buffer, body)
	}
	if err != nil {
		return err
	}
	m.bytes = buffer.Bytes()
	m.size = uint32(len(m.bytes))
	return nil
}

// Unmarshal 解析Message body
func (m *message) Unmarshal(i any) (err error) {
	meta := m.Head.Query()
	bi := binder.GetContentType(meta, binder.ContentTypeModRes)
	return bi.Unmarshal(m.Body(), i)
}

func (m *message) Release() {
	m.Head.Release()
}

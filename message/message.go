package message

import (
	"bytes"
	"errors"
	"io"
	"strings"
)

type message struct {
	Head
	bytes []byte //数据
}

func (m *message) Path() (r, q string, err error) {
	magic := m.Magic()
	if magic.Type == MagicTypePath {
		if int(m.code) > len(m.bytes) {
			err = ErrMsgHeadIllegal
			return
		}
		r = string(m.bytes[0:int(m.code)])
		if i := strings.Index(r, "?"); i >= 0 {
			r = r[:i]
			q = r[i+1:]
		}
	} else {
		r, err = Transform.Path(m.code)
	}
	return
}

// Body 消息体
func (m *message) Body() []byte {
	magic := m.Magic()
	if magic.Type == MagicTypePath {
		return m.bytes[int(m.code):]
	} else {
		return m.bytes
	}
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
func (m *message) Marshal(magic byte, index int32, path string, body any) (err error) {
	if err = m.Head.format(magic, index, path); err != nil {
		return
	}
	mc := m.Magic()
	buffer := bytes.NewBuffer(m.bytes[0:0])

	if mc.Type == MagicTypePath {
		buffer.WriteString(path)
	}
	switch v := body.(type) {
	case []byte:
		if len(v) > 0 {
			buffer.Write(v)
		}
	default:
		bi := m.Binder()
		if bi == nil {
			err = errors.New("no binder found")
		} else {
			err = bi.Encode(buffer, body)
		}
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
	bi := m.Binder()
	if bi == nil {
		return errors.New("no binder found")
	}
	return bi.Unmarshal(m.Body(), i)
}

func (m *message) Release() {
	m.Head.Release()
}

package wss

import (
	"bytes"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/hwcer/cosnet/listener"
	"github.com/hwcer/cosnet/message"
)

// Options WSS模块配置选项
var Options = struct {
	Origin       []string
	ConnChanSize int32
	Upgrader     websocket.Upgrader
	Transform    transform
}{
	Origin:       []string{},                                                      // 默认允许所有来源
	ConnChanSize: 100,                                                             // 连接通道缓存 100 条消息
	Upgrader:     websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}, // 默认配置，允许所有来源
	Transform:    defaultTransform{},
}

func init() {
	Options.Upgrader.CheckOrigin = AccessControlAllow
}

func AccessControlAllow(r *http.Request) bool {
	if len(Options.Origin) == 0 {
		return true
	}
	for _, o := range Options.Origin {
		if o == "*" || o == r.URL.Host {
			return true
		}
	}
	return false
}

type transform interface {
	ReadMessage(socket listener.Socket, msg message.Message, data []byte) error
	WriteMessage(socket listener.Socket, msg message.Message, b *bytes.Buffer) (n int, err error)
}

type defaultTransform struct{}

func (t defaultTransform) ReadMessage(_ listener.Socket, m message.Message, data []byte) error {
	return m.Reset(data)
}
func (t defaultTransform) WriteMessage(_ listener.Socket, msg message.Message, b *bytes.Buffer) (n int, err error) {
	return msg.Bytes(b, true)
}

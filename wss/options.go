package wss

import (
	"bytes"
	"net/http"
	"net/url"

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
	//🔴 白名单必须比对 Origin 请求头:旧实现比对 r.URL.Host——浏览器 WS 升级请求行
	//是 origin-form,该值几乎恒为空串,白名单永不命中(正常跨域客户端全被拒),
	//伪造来源防护也完全没做
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	host := origin
	if u, err := url.Parse(origin); err == nil {
		host = u.Host
	}
	for _, o := range Options.Origin {
		if o == "*" || o == origin || o == host {
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

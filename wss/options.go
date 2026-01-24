package wss

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// Options WSS模块配置选项
var Options = struct {
	// Origin 允许的来源域名列表
	Origin []string
	// ConnChanSize 连接通道缓存大小
	ConnChanSize int32
	// Upgrader websocket升级器配置
	Upgrader websocket.Upgrader
}{
	Origin:       []string{},                                                      // 默认允许所有来源
	ConnChanSize: 100,                                                             // 连接通道缓存 100 条消息
	Upgrader:     websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024}, // 默认配置，允许所有来源
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

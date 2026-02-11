package pubsub

// 订阅相关的消息路径
const (
	PathSubscribe     = "pubsub/subscribe"     // 订阅主题
	PathUnsubscribe   = "pubsub/unsubscribe"   // 取消订阅
	PathPublish       = "pubsub/publish"       // 发布消息
	PathSubscribeList = "pubsub/subscribe/list" // 获取订阅列表
	PathMessage       = "pubsub/message"       // 接收发布的消息
)

// SubscribeData 订阅数据
type SubscribeData struct {
	Topics []string `json:"topics"` // 主题列表
}

// UnsubscribeData 取消订阅数据
type UnsubscribeData struct {
	Topics []string `json:"topics"` // 主题列表
}

// PublishData 发布数据
type PublishData struct {
	Topic   string      `json:"topic"`   // 主题
	Message interface{} `json:"message"` // 消息内容
}

// SubscribeListData 订阅列表数据
type SubscribeListData struct {
	Topics []string `json:"topics"` // 订阅的主题列表
}

// MessageData 发布消息数据
type MessageData struct {
	Topic   string      `json:"topic"`   // 主题
	Message interface{} `json:"message"` // 消息内容
}

// ResponseData 响应数据
type ResponseData struct {
	Code    string      `json:"code"`    // 响应码
	Message string      `json:"message"` // 响应消息
	Data    interface{} `json:"data"`    // 响应数据
}

package pubsub

import (
	"github.com/hwcer/cosnet"
)

// Handler 订阅发布系统处理器
type Handler struct{}

// Subscribe 处理订阅请求
func (h *Handler) Subscribe(c *cosnet.Context) interface{} {
	var data SubscribeData
	if err := c.Bind(&data); err != nil {
		return ResponseData{
			Code:    "invalid_data",
			Message: "无效的数据格式",
		}
	}

	for _, topic := range data.Topics {
		PubSub.Subscribe(c.Socket, topic)
	}

	return ResponseData{
		Code:    "subscribed",
		Message: "订阅成功",
	}
}

// Unsubscribe 处理取消订阅请求
func (h *Handler) Unsubscribe(c *cosnet.Context) interface{} {
	var data UnsubscribeData
	if err := c.Bind(&data); err != nil {
		return ResponseData{
			Code:    "invalid_data",
			Message: "无效的数据格式",
		}
	}

	for _, topic := range data.Topics {
		PubSub.Unsubscribe(c.Socket, topic)
	}

	return ResponseData{
		Code:    "unsubscribed",
		Message: "取消订阅成功",
	}
}

// Publish 处理发布请求
func (h *Handler) Publish(c *cosnet.Context) interface{} {
	var data PublishData
	if err := c.Bind(&data); err != nil {
		return ResponseData{
			Code:    "invalid_data",
			Message: "无效的数据格式",
		}
	}

	PubSub.Publish(data.Topic, data.Message)
	return ResponseData{
		Code:    "published",
		Message: "发布成功",
	}
}

// SubscribeList 处理获取订阅列表请求
func (h *Handler) SubscribeList(c *cosnet.Context) interface{} {
	topics := PubSub.GetSubscriptions(c.Socket)
	return ResponseData{
		Code:    "subscriptions",
		Message: "获取订阅列表成功",
		Data:    SubscribeListData{Topics: topics},
	}
}

// Init 初始化订阅发布系统
func Init() {
	// 注册处理器
	cosnet.Register(&Handler{}, "pubsub/")

	// 注册连接断开事件，自动取消所有订阅
	cosnet.On(cosnet.EventTypeDisconnect, func(socket *cosnet.Socket, _ any) {
		PubSub.UnsubscribeAll(socket)
	})
}

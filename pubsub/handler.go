package pubsub

import (
	"github.com/hwcer/cosnet"
)

// handler 订阅发布系统处理器
type handler struct {
	pb *PubSub
}

// Subscribe 处理订阅请求
func (h *handler) Subscribe(c *cosnet.Context) interface{} {
	var data SubscribeData
	if err := c.Bind(&data); err != nil {
		return ResponseData{
			Code:    "invalid_data",
			Message: "无效的数据格式",
		}
	}

	for _, topic := range data.Topics {
		h.pubsub.Subscribe(c.Socket, topic, nil)
	}

	return ResponseData{
		Code:    "subscribed",
		Message: "订阅成功",
	}
}

// QueueSubscribe 处理队列订阅请求
func (h *handler) QueueSubscribe(c *cosnet.Context) interface{} {
	var data QueueSubscribeData
	if err := c.Bind(&data); err != nil {
		return ResponseData{
			Code:    "invalid_data",
			Message: "无效的数据格式",
		}
	}

	for _, topic := range data.Topics {
		h.pubsub.QueueSubscribe(c.Socket, topic, data.Queue, nil)
	}

	return ResponseData{
		Code:    "queue_subscribed",
		Message: "队列订阅成功",
	}
}

// Unsubscribe 处理取消订阅请求
func (h *handler) Unsubscribe(c *cosnet.Context) interface{} {
	var data UnsubscribeData
	if err := c.Bind(&data); err != nil {
		return ResponseData{
			Code:    "invalid_data",
			Message: "无效的数据格式",
		}
	}

	for _, topic := range data.Topics {
		h.pubsub.Unsubscribe(c.Socket, topic)
	}

	return ResponseData{
		Code:    "unsubscribed",
		Message: "取消订阅成功",
	}
}

// Publish 处理发布请求
func (h *handler) Publish(c *cosnet.Context) interface{} {
	var data PublishData
	if err := c.Bind(&data); err != nil {
		return ResponseData{
			Code:    "invalid_data",
			Message: "无效的数据格式",
		}
	}

	h.pubsub.Publish(data.Topic, data.Message)
	return ResponseData{
		Code:    "published",
		Message: "发布成功",
	}
}

// Request 处理请求响应
func (h *handler) Request(c *cosnet.Context) interface{} {
	var data RequestData
	if err := c.Bind(&data); err != nil {
		return ResponseData{
			Code:    "invalid_data",
			Message: "无效的数据格式",
		}
	}

	response := h.pubsub.Request(data.Topic, data.Message, data.Timeout)
	return ResponseData{
		Code:    "requested",
		Message: "请求发送成功",
		Data:    response,
	}
}

// SubscribeList 处理获取订阅列表请求
func (h *handler) SubscribeList(c *cosnet.Context) interface{} {
	topics := h.pubsub.GetSubscriptions(c.Socket)
	return ResponseData{
		Code:    "subscriptions",
		Message: "获取订阅列表成功",
		Data:    SubscribeListData{Topics: topics},
	}
}

// Init 初始化订阅发布系统
func Init() {
	// 创建默认的PubSub实例
	ps := New(30)
	handler := NewHandler(ps)

	// 注册处理器
	ps.Register(handler, "pubsub/")

	// 注册连接断开事件，自动取消所有订阅
	ps.On(cosnet.EventTypeDisconnect, func(socket *cosnet.Socket, _ any) {
		ps.UnsubscribeAll(socket)
	})
}

// InitWithCosnet 使用指定的cosnet实例初始化订阅发布系统
func InitWithCosnet(cos *cosnet.NetHub) *PubSub {
	// 创建PubSub实例
	ps := NewWithCosnet(cos)
	handler := NewHandler(ps)

	// 注册处理器
	ps.Register(handler, "pubsub/")

	// 注册连接断开事件，自动取消所有订阅
	ps.On(cosnet.EventTypeDisconnect, func(socket *cosnet.Socket, _ any) {
		ps.UnsubscribeAll(socket)
	})

	return ps
}

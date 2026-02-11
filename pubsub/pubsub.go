package pubsub

import (
	"sync"

	"github.com/hwcer/cosnet/message"
)

// Socket 接口定义，用于与 cosnet 的 Socket 结构兼容
type Socket interface {
	Id() uint64
	Alive() bool
	Send(flag message.Flag, index int32, path string, data any)
}

// PubSub 订阅发布系统
var PubSub = &pubsub{
	subscriptions: make(map[string]map[uint64]Socket),
	mutex:         &sync.RWMutex{},
}

// pubsub 订阅发布系统实现
type pubsub struct {
	subscriptions map[string]map[uint64]Socket // 主题 -> 订阅者映射
	mutex         *sync.RWMutex                // 并发锁
}

// Subscribe 订阅主题
func (ps *pubsub) Subscribe(socket Socket, topic string) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if _, ok := ps.subscriptions[topic]; !ok {
		ps.subscriptions[topic] = make(map[uint64]Socket)
	}

	ps.subscriptions[topic][socket.Id()] = socket
}

// Unsubscribe 取消订阅主题
func (ps *pubsub) Unsubscribe(socket Socket, topic string) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if subscribers, ok := ps.subscriptions[topic]; ok {
		delete(subscribers, socket.Id())

		// 如果主题没有订阅者，删除主题
		if len(subscribers) == 0 {
			delete(ps.subscriptions, topic)
		}
	}
}

// UnsubscribeAll 取消所有订阅
func (ps *pubsub) UnsubscribeAll(socket Socket) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	for topic, subscribers := range ps.subscriptions {
		if _, ok := subscribers[socket.Id()]; ok {
			delete(subscribers, socket.Id())

			// 如果主题没有订阅者，删除主题
			if len(subscribers) == 0 {
				delete(ps.subscriptions, topic)
			}
		}
	}
}

// Publish 发布消息到主题
func (ps *pubsub) Publish(topic string, msg interface{}) {
	ps.mutex.RLock()
	subscribers, ok := ps.subscriptions[topic]
	ps.mutex.RUnlock()

	if !ok || len(subscribers) == 0 {
		return
	}

	// 向所有订阅者发送消息
	for _, socket := range subscribers {
		if socket.Alive() {
			socket.Send(0, 0, PathMessage, MessageData{
				Topic:   topic,
				Message: msg,
			})
		}
	}
}

// GetSubscriptions 获取socket的订阅列表
func (ps *pubsub) GetSubscriptions(socket Socket) []string {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	var topics []string
	for topic, subscribers := range ps.subscriptions {
		if _, ok := subscribers[socket.Id()]; ok {
			topics = append(topics, topic)
		}
	}

	return topics
}

// GetSubscriberCount 获取主题的订阅者数量
func (ps *pubsub) GetSubscriberCount(topic string) int {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	if subscribers, ok := ps.subscriptions[topic]; ok {
		return len(subscribers)
	}

	return 0
}

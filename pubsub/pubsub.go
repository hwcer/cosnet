package pubsub

import (
	"regexp"
	"sync"

	"github.com/hwcer/cosnet"
	"github.com/hwcer/cosnet/listener"
)

// Subscription 表示一个订阅
type Subscription struct {
	Topic   string
	Queue   string
	Handler func(topic string, msg any)
}

type NetSubscription struct {
	Subscription
	Socket listener.Socket
}

// PubSub 订阅发布系统
type PubSub struct {
	*cosnet.NetHub
	local         map[string]*Subscription               //本地订阅（客户端功能）
	subscriptions map[string]map[uint64]*NetSubscription //远程订阅（服务器功能）
	mutex         *sync.RWMutex
}

// New 创建一个新的PubSub实例
func New(heartbeat int32) *PubSub {
	hub := cosnet.New(heartbeat)
	return NewWithNetHub(hub)
}

// NewWithNetHub 使用指定的cosnet实例创建PubSub
func NewWithNetHub(hub *cosnet.NetHub) *PubSub {
	pb := &PubSub{
		NetHub:        hub,
		local:         make(map[string]*Subscription),
		subscriptions: make(map[string]map[uint64]*NetSubscription),
		mutex:         &sync.RWMutex{},
	}
	handle := new(handler)
	_ = pb.Register(handle, BasePath, "%m")
	return pb
}

// Subscribe 订阅主题
func (ps *PubSub) Subscribe(topic string, handler func(topic string, msg interface{})) {
	ps.mutex.Lock()
	if ps.local == nil {
		ps.local = make(map[string]*Subscription)
	}
	v := &Subscription{Topic: topic, Handler: handler}
	if _, ok := ps.local[topic]; !ok {
		ps.local[topic] = v
	}
	ps.mutex.Unlock()
	//发送给所有服务器
	ps.Range(func(socket *cosnet.Socket) bool {
		if socket.Type() != listener.SocketTypeClient {
			return true
		}
		socket.Send(0, 0, PathSubscribe, v)
		return true
	})

}

// Unsubscribe 取消订阅主题
func (ps *PubSub) Unsubscribe(topic string) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	if subscribers, ok := ps.subscriptions[topic]; ok {
		delete(subscribers, socket.Id())

		if len(subscribers) == 0 {
			delete(ps.subscriptions, topic)
		}
	}
}

// UnsubscribeAll 取消所有订阅
func (ps *PubSub) UnsubscribeAll(socket listener.Socket) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	for topic, subscribers := range ps.subscriptions {
		if _, ok := subscribers[socket.Id()]; ok {
			delete(subscribers, socket.Id())

			if len(subscribers) == 0 {
				delete(ps.subscriptions, topic)
			}
		}
	}
}

// Publish 发布消息到主题
func (ps *PubSub) Publish(topic string, msg interface{}) {
	ps.mutex.RLock()
	subscribers, ok := ps.subscriptions[topic]
	wildcardSubscribers := ps.getWildcardSubscribers(topic)
	ps.mutex.RUnlock()

	if !ok && len(wildcardSubscribers) == 0 {
		return
	}

	if ok && len(subscribers) > 0 {
		ps.sendToSubscribers(subscribers, topic, msg)
	}

	if len(wildcardSubscribers) > 0 {
		ps.sendToSubscribers(wildcardSubscribers, topic, msg)
	}
}

// getWildcardSubscribers 获取匹配通配符的订阅者
func (ps *PubSub) getWildcardSubscribers(topic string) map[uint64]*Subscription {
	result := make(map[uint64]*Subscription)

	for subTopic, subscribers := range ps.subscriptions {
		if ps.matchesWildcard(subTopic, topic) {
			for id, sub := range subscribers {
				result[id] = sub
			}
		}
	}

	return result
}

// matchesWildcard 检查主题是否匹配通配符
func (ps *PubSub) matchesWildcard(subTopic, topic string) bool {
	rePattern := subTopic
	rePattern = regexp.QuoteMeta(rePattern)
	rePattern = regexp.MustCompile(`\\\*`).ReplaceAllString(rePattern, `[^.]+`)
	rePattern = regexp.MustCompile(`\\\>`).ReplaceAllString(rePattern, `.+`)
	rePattern = `^` + rePattern + `$`

	matched, _ := regexp.MatchString(rePattern, topic)
	return matched
}

// sendToSubscribers 向订阅者发送消息
func (ps *PubSub) sendToSubscribers(subscribers map[uint64]*Subscription, topic string, msg interface{}) {
	queueGroups := make(map[string][]uint64)
	for id, sub := range subscribers {
		if sub.Queue != "" {
			queueGroups[sub.Queue] = append(queueGroups[sub.Queue], id)
		}
	}

	for _, ids := range queueGroups {
		if len(ids) > 0 {
			if sub, ok := subscribers[ids[0]]; ok {
				if sub.Handler != nil {
					sub.Handler(topic, msg)
				} else {
					sub.Socket.Send(0, 0, PathMessage, MessageData{
						Topic:   topic,
						Message: msg,
					})
				}
			}
		}
	}

	for _, sub := range subscribers {
		if sub.Queue == "" {
			if sub.Handler != nil {
				sub.Handler(topic, msg)
			} else {
				sub.Socket.Send(0, 0, PathMessage, MessageData{
					Topic:   topic,
					Message: msg,
				})
			}
		}
	}
}

// GetSubscriptions 获取socket的订阅列表
func (ps *PubSub) GetSubscriptions(socket listener.Socket) []string {
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
func (ps *PubSub) GetSubscriberCount(topic string) int {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	if subscribers, ok := ps.subscriptions[topic]; ok {
		return len(subscribers)
	}

	return 0
}

// Request 发送请求并等待响应
func (ps *PubSub) Request(topic string, msg interface{}, timeout int) interface{} {
	ps.Publish(topic, msg)
	return nil
}

// Flush 刷新所有消息
func (ps *PubSub) Flush() {
}

// Close 关闭发布订阅系统
func (ps *PubSub) Close() {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.subscriptions = make(map[string]map[uint64]*Subscription)
}

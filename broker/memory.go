package broker

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
)

// MemoryBroker 是 broker.Broker 的进程内参考实现，主要用于测试与示例。
// 不持久化、不跨进程、不保证消息顺序在多订阅者之间一致。
type MemoryBroker struct {
	opts Options

	mu          sync.RWMutex
	connected   bool
	subscribers map[string]map[string]*memorySubscription // topic -> subID -> sub
}

// NewMemory 构造一个进程内内存 broker，应用 Wrappers 链。
func NewMemory(opts ...Option) Broker {
	o := NewOptions(opts...)
	b := &MemoryBroker{
		opts:        o,
		subscribers: make(map[string]map[string]*memorySubscription),
	}
	var br Broker = b
	for i := len(o.Wrappers) - 1; i >= 0; i-- {
		br = o.Wrappers[i](br)
	}
	return br
}

// Connect 建立内存连接（实际只是切换状态）。
func (m *MemoryBroker) Connect(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}

// Disconnect 关闭所有订阅并切换为未连接状态。
func (m *MemoryBroker) Disconnect(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.connected {
		return nil
	}
	for topic, subs := range m.subscribers {
		for id, sub := range subs {
			sub.markClosed()
			delete(subs, id)
		}
		delete(m.subscribers, topic)
	}
	m.connected = false
	return nil
}

// Publish 同步将消息分发给所有订阅 topic 的 handler。
// 多个订阅者时：
//   - 同一 Queue 组内随机选一个投递（实现为按 map 遍历的第一个）
//   - 不同 Queue 的订阅者都会收到
func (m *MemoryBroker) Publish(ctx context.Context, topic string, msg *Message, opts ...PublishOption) error {
	m.mu.RLock()
	if !m.connected {
		m.mu.RUnlock()
		return ErrNotConnected
	}
	bucket := m.subscribers[topic]
	// 复制订阅者引用，避免持锁调用 handler
	subs := make([]*memorySubscription, 0, len(bucket))
	for _, s := range bucket {
		subs = append(subs, s)
	}
	m.mu.RUnlock()

	if len(subs) == 0 {
		return nil
	}

	po := NewPublishOptions(opts...)
	// 合并 publish 时携带的 header 到消息上
	if len(po.Header) > 0 {
		if msg.Header == nil {
			msg.Header = make(Header, len(po.Header))
		}
		for k, v := range po.Header {
			msg.Header[k] = v
		}
	}

	// 按 Queue 分组：同组只投递一个
	delivered := make(map[string]bool)
	for _, s := range subs {
		key := s.options.Queue
		if key != "" {
			if delivered[key] {
				continue
			}
			delivered[key] = true
		}
		evt := &memoryEvent{topic: topic, msg: msg}
		if err := s.handler(ctx, evt); err != nil {
			if !s.options.AutoAck {
				// Handler 已自行决定 Ack/Nack
				continue
			}
			// AutoAck 模式下错误意味着失败：调用 Nack
			_ = evt.Nack(err)
			continue
		}
		if s.options.AutoAck {
			_ = evt.Ack()
		}
	}
	return nil
}

// Subscribe 注册一个订阅。返回的 Subscription 可用于取消。
func (m *MemoryBroker) Subscribe(_ context.Context, topic string, handler Handler, opts ...SubscribeOption) (Subscription, error) {
	o := NewSubscribeOptions(opts...)

	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.connected {
		return nil, ErrNotConnected
	}
	bucket, ok := m.subscribers[topic]
	if !ok {
		bucket = make(map[string]*memorySubscription)
		m.subscribers[topic] = bucket
	}

	sub := &memorySubscription{
		id:      uuid.New().String(),
		topic:   topic,
		handler: handler,
		options: o,
		broker:  m,
	}
	bucket[sub.id] = sub
	return sub, nil
}

// ErrNotConnected 在尚未 Connect 时操作 broker 返回。
var ErrNotConnected = errors.New("broker: not connected")

// ============================================================
// 内部类型
// ============================================================

type memorySubscription struct {
	id      string
	topic   string
	handler Handler
	options SubscribeOptions
	broker  *MemoryBroker

	mu     sync.Mutex
	closed bool
}

func (s *memorySubscription) Topic() string { return s.topic }

func (s *memorySubscription) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.mu.Unlock()

	s.broker.mu.Lock()
	defer s.broker.mu.Unlock()
	if bucket, ok := s.broker.subscribers[s.topic]; ok {
		delete(bucket, s.id)
		if len(bucket) == 0 {
			delete(s.broker.subscribers, s.topic)
		}
	}
	return nil
}

func (s *memorySubscription) markClosed() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
}

type memoryEvent struct {
	topic string
	msg   *Message

	mu      sync.Mutex
	acked   bool
	nackErr error
}

func (e *memoryEvent) Topic() string    { return e.topic }
func (e *memoryEvent) Message() *Message { return e.msg }

func (e *memoryEvent) Ack() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.acked {
		return nil
	}
	e.acked = true
	return nil
}

func (e *memoryEvent) Nack(err error) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.nackErr = err
	return nil
}

// ensure interfaces
var (
	_ Broker       = (*MemoryBroker)(nil)
	_ Subscription = (*memorySubscription)(nil)
	_ Event        = (*memoryEvent)(nil)
)

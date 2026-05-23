// Package typed 在 broker.Broker（[]byte 层）之上提供基于 Go 泛型的类型安全 API。
//
// 核心抽象：
//
//   - Topic[T]   ：声明式主题，把主题名称与 payload 类型绑定
//   - Codec[T]   ：负责 T 与 []byte 的编解码
//   - Publisher[T] ：发布 T 类型的消息
//   - Subscriber[T] ：订阅 T 类型的消息
//   - Handler[T]   ：处理 T 类型消息的回调
//   - Event[T]     ：包含类型化 payload 的消息事件
//
// 用法：
//
//	var OrderCreated = typed.Topic[*pb.Order]{
//	    Name:  "orders.created",
//	    Codec: typed.ProtoCodec[*pb.Order]{},
//	}
//
//	pub := typed.NewPublisher(rawBroker, OrderCreated)
//	_ = pub.Publish(ctx, &pb.Order{ID: "123"})
//
//	sub := typed.NewSubscriber(rawBroker, OrderCreated)
//	_, _ = sub.Subscribe(ctx, func(ctx context.Context, e typed.Event[*pb.Order]) error {
//	    order := e.Payload() // 已经是 *pb.Order，无需 unmarshal
//	    return process(order)
//	})
package typed

import (
	"context"
	"fmt"

	"github.com/nextmicro/next/broker"
)

// Codec[T] 负责 T 类型与 []byte 的编解码。
//
// 实现需保证：
//   - Marshal 后立即 Unmarshal 应得到等价值（roundtrip 一致性）
//   - Unmarshal 失败必须返回错误而非 panic
type Codec[T any] interface {
	// Marshal 将 v 序列化为字节流。
	Marshal(v T) ([]byte, error)
	// Unmarshal 将字节流反序列化为 T。
	Unmarshal(data []byte, out *T) error
	// ContentType 返回媒体类型，例如 "application/x-protobuf"、"application/json"。
	ContentType() string
}

// Topic[T] 是带类型标签的主题声明。
// 将主题名称与 payload 类型在编译期绑定。
type Topic[T any] struct {
	// Name 是底层 broker 主题名称。
	Name string

	// Codec 负责 T 的编解码。
	Codec Codec[T]
}

// Publisher[T] 是单一主题的类型安全发布器。
type Publisher[T any] interface {
	// Publish 将类型化消息编码并发布到 topic。
	Publish(ctx context.Context, msg T, opts ...broker.PublishOption) error
}

// Subscriber[T] 是单一主题的类型安全订阅器。
type Subscriber[T any] interface {
	// Subscribe 订阅 topic，使用类型化 handler 处理消息。
	Subscribe(ctx context.Context, handler Handler[T], opts ...broker.SubscribeOption) (broker.Subscription, error)
}

// Handler[T] 处理类型化消息。
type Handler[T any] func(ctx context.Context, e Event[T]) error

// Event[T] 是 typed 层的消息事件。
type Event[T any] interface {
	// Topic 返回消息主题。
	Topic() string

	// Payload 返回已反序列化的类型化负载。
	Payload() T

	// Header 返回原始消息头部。
	Header() broker.Header

	// Ack 显式确认。
	Ack() error

	// Nack 显式拒绝。
	Nack(err error) error
}

// ============================================================
// 默认 Publisher / Subscriber 实现
// ============================================================

type publisher[T any] struct {
	broker broker.Broker
	topic  Topic[T]
}

// NewPublisher 基于底层 Broker 与 Topic 创建类型安全 Publisher。
// Topic.Codec 不可为 nil，否则返回的 Publisher 在 Publish 时会返回错误。
func NewPublisher[T any](b broker.Broker, t Topic[T]) Publisher[T] {
	return &publisher[T]{broker: b, topic: t}
}

func (p *publisher[T]) Publish(ctx context.Context, msg T, opts ...broker.PublishOption) error {
	if p.topic.Codec == nil {
		return fmt.Errorf("typed: topic %q has no codec", p.topic.Name)
	}
	body, err := p.topic.Codec.Marshal(msg)
	if err != nil {
		return fmt.Errorf("typed: marshal topic %q: %w", p.topic.Name, err)
	}
	po := broker.NewPublishOptions(opts...)

	m := &broker.Message{Body: body}
	if po.Header != nil {
		m.Header = po.Header
	}
	if ct := p.topic.Codec.ContentType(); ct != "" {
		if m.Header == nil {
			m.Header = make(broker.Header)
		}
		if _, exists := m.Header["Content-Type"]; !exists {
			m.Header["Content-Type"] = ct
		}
	}
	return p.broker.Publish(ctx, p.topic.Name, m, opts...)
}

type subscriber[T any] struct {
	broker broker.Broker
	topic  Topic[T]
}

// NewSubscriber 基于底层 Broker 与 Topic 创建类型安全 Subscriber。
func NewSubscriber[T any](b broker.Broker, t Topic[T]) Subscriber[T] {
	return &subscriber[T]{broker: b, topic: t}
}

func (s *subscriber[T]) Subscribe(ctx context.Context, handler Handler[T], opts ...broker.SubscribeOption) (broker.Subscription, error) {
	if s.topic.Codec == nil {
		return nil, fmt.Errorf("typed: topic %q has no codec", s.topic.Name)
	}

	rawHandler := func(ctx context.Context, e broker.Event) error {
		var payload T
		m := e.Message()
		if m == nil {
			return fmt.Errorf("typed: nil message on topic %q", s.topic.Name)
		}
		if err := s.topic.Codec.Unmarshal(m.Body, &payload); err != nil {
			nackErr := fmt.Errorf("typed: unmarshal topic %q: %w", s.topic.Name, err)
			// 尝试 Nack 让底层 broker 知道消息消费失败
			if nerr := e.Nack(nackErr); nerr != nil {
				return fmt.Errorf("%w (nack failed: %v)", nackErr, nerr)
			}
			return nackErr
		}
		typedEvent := &event[T]{raw: e, payload: payload}
		return handler(ctx, typedEvent)
	}
	return s.broker.Subscribe(ctx, s.topic.Name, rawHandler, opts...)
}

// event 是 Event[T] 的默认实现，包装一个 broker.Event。
type event[T any] struct {
	raw     broker.Event
	payload T
}

func (e *event[T]) Topic() string         { return e.raw.Topic() }
func (e *event[T]) Payload() T            { return e.payload }
func (e *event[T]) Header() broker.Header { return e.raw.Message().Header }
func (e *event[T]) Ack() error            { return e.raw.Ack() }
func (e *event[T]) Nack(err error) error  { return e.raw.Nack(err) }

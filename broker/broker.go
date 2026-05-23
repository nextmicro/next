// Package broker 提供异步消息通信抽象。
//
// 设计分层：
//
//   - 底层 broker.Broker 处理 []byte 消息，保持插件化能力
//     （memory / kafka / nats / 等多种实现可插拔）。
//
//   - 上层 broker/typed 包提供基于 Go 泛型的类型安全 Publisher[T] /
//     Subscriber[T]，业务代码无需手动 marshal / unmarshal。
//
// 用法示例：
//
//	// 底层：直接发布字节
//	b := memory.New()
//	_ = b.Connect(ctx)
//	_ = b.Publish(ctx, "topic", &broker.Message{Body: []byte("hello")})
//
//	// 上层：类型安全的 Publisher
//	import "github.com/nextmicro/next/broker/typed"
//
//	var OrderCreated = typed.Topic[*pb.Order]{
//	    Name:  "orders.created",
//	    Codec: typed.ProtoCodec[*pb.Order]{},
//	}
//	pub := typed.NewPublisher(b, OrderCreated)
//	_ = pub.Publish(ctx, &pb.Order{ID: "123"})
package broker

import "context"

// Broker 是底层异步消息中间件抽象。
//
// 接口设计原则：
//   - 仅暴露连接管理（Connect/Disconnect）与字节级消息传输（Publish/Subscribe）
//   - 不暴露 Init/Options/Address/String 等内部状态方法
//   - 配置通过构造函数注入，不支持运行时再次 Init
type Broker interface {
	// Connect 建立到底层消息系统的连接。
	// 实现应在 Connect 中完成所有需要建立连接的资源准备。
	// 重复 Connect 应是幂等的。
	Connect(ctx context.Context) error

	// Disconnect 释放底层连接与相关资源。
	// 重复 Disconnect 应是幂等的，且必须中止所有进行中的 Subscribe。
	Disconnect(ctx context.Context) error

	// Publish 向 topic 发送消息。
	// 消息体在 Message.Body 中，可选 Message.Header 携带元数据。
	// 返回错误时调用方应自行决定重试策略。
	Publish(ctx context.Context, topic string, msg *Message, opts ...PublishOption) error

	// Subscribe 订阅 topic 并通过 handler 异步消费消息。
	// 返回的 Subscription 可用于取消订阅，且仅影响该订阅，不影响其他订阅或连接。
	Subscribe(ctx context.Context, topic string, handler Handler, opts ...SubscribeOption) (Subscription, error)
}

// Handler 是底层消息处理函数。
// 实现可调用 event.Ack() / event.Nack(err) 显式确认或拒绝。
// 返回 nil 时框架按 SubscribeOptions.AutoAck 决定是否自动 Ack。
type Handler func(ctx context.Context, event Event) error

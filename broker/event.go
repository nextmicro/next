package broker

// Event 是订阅 Handler 收到的消息载体。
// 提供主题信息、原始消息、确认/拒绝能力。
type Event interface {
	// Topic 返回此消息所属的主题。
	Topic() string

	// Message 返回原始字节级消息。
	Message() *Message

	// Ack 显式确认消息已成功处理。
	// 在 AutoAck=true 模式下，Handler 返回 nil 时框架自动调用。
	Ack() error

	// Nack 显式拒绝消息（处理失败）。
	// 实现根据底层 broker 能力决定行为：
	//   - 重新投递（kafka 重平衡 / nats redelivery）
	//   - 投入死信队列
	//   - 仅记录指标
	Nack(err error) error
}

package broker

// Subscription 表示一个活跃的订阅。
// 关闭 Subscription 仅终止该订阅，不影响同一 Broker 的其他订阅或发布。
type Subscription interface {
	// Topic 返回此订阅监听的主题。
	Topic() string

	// Close 终止订阅。
	// 实现应是幂等的：重复 Close 不应返回错误。
	Close() error
}

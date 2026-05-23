package broker

import (
	"context"
	"crypto/tls"
)

// Options 是 Broker 构造期的配置。
// 各 broker 实现（memory/kafka/...）的 New 函数都接受 ...Option。
type Options struct {
	// Addrs 是底层消息系统的地址列表。
	Addrs []string

	// TLSConfig 启用 TLS 时使用的配置；nil 表示明文连接。
	TLSConfig *tls.Config

	// Secure 是否使用安全连接（与具体实现协商）。
	Secure bool

	// Wrappers 是按声明顺序应用的装饰器链。
	// 最常用于 logging / metrics / tracing 横切能力的注入。
	Wrappers []Wrapper

	// Context 用于派生 broker 内部 goroutine 的根上下文。
	Context context.Context
}

// Option 配置 Broker 的构造参数。
type Option func(*Options)

// NewOptions 应用 opts 构造 Options。
func NewOptions(opts ...Option) Options {
	o := Options{Context: context.Background()}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// Addrs 设置 broker 地址列表。
func Addrs(addrs ...string) Option {
	return func(o *Options) { o.Addrs = addrs }
}

// TLSConfig 设置 TLS 配置。
func TLSConfig(c *tls.Config) Option {
	return func(o *Options) { o.TLSConfig = c }
}

// Secure 设置是否使用安全连接。
func Secure(b bool) Option {
	return func(o *Options) { o.Secure = b }
}

// Wrap 追加 Wrapper 到链中。多次调用会累计。
func Wrap(w ...Wrapper) Option {
	return func(o *Options) { o.Wrappers = append(o.Wrappers, w...) }
}

// WithContext 设置内部 goroutine 的根上下文。
func WithContext(ctx context.Context) Option {
	return func(o *Options) { o.Context = ctx }
}

// ============================================================
// Publish 选项
// ============================================================

// PublishOptions 是单次发布的选项。
type PublishOptions struct {
	// Header 附加到本次消息的额外头部，会与 Message.Header 合并。
	Header Header

	// Context 覆盖发布操作使用的上下文；为 nil 时使用 Publish 调用的 ctx。
	Context context.Context
}

// PublishOption 配置单次 Publish 调用。
type PublishOption func(*PublishOptions)

// NewPublishOptions 应用 opts 构造 PublishOptions。
func NewPublishOptions(opts ...PublishOption) PublishOptions {
	o := PublishOptions{}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// WithPublishHeader 为本次发布追加一个头部 key/value。
func WithPublishHeader(key, value string) PublishOption {
	return func(o *PublishOptions) {
		if o.Header == nil {
			o.Header = make(Header)
		}
		o.Header[key] = value
	}
}

// WithPublishHeaders 一次性设置/覆盖多个头部。
func WithPublishHeaders(h Header) PublishOption {
	return func(o *PublishOptions) {
		if o.Header == nil {
			o.Header = make(Header, len(h))
		}
		for k, v := range h {
			o.Header[k] = v
		}
	}
}

// WithPublishContext 覆盖发布操作的上下文。
func WithPublishContext(ctx context.Context) PublishOption {
	return func(o *PublishOptions) { o.Context = ctx }
}

// ============================================================
// Subscribe 选项
// ============================================================

// SubscribeOptions 是订阅时的选项。
type SubscribeOptions struct {
	// Queue 是订阅者所属的"队列组"。同组成员共享订阅，
	// 每条消息仅投递给组内一个成员（具体实现可能不支持）。
	Queue string

	// AutoAck 控制 Handler 返回 nil 时是否自动 Ack。默认 true。
	AutoAck bool

	// Context 覆盖订阅期间使用的上下文。
	Context context.Context
}

// SubscribeOption 配置 Subscribe 调用。
type SubscribeOption func(*SubscribeOptions)

// NewSubscribeOptions 应用 opts，并填充默认值（AutoAck=true）。
func NewSubscribeOptions(opts ...SubscribeOption) SubscribeOptions {
	o := SubscribeOptions{AutoAck: true}
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// WithSubscribeQueue 设置订阅所属的队列组。
func WithSubscribeQueue(name string) SubscribeOption {
	return func(o *SubscribeOptions) { o.Queue = name }
}

// WithSubscribeAutoAck 控制是否自动 Ack。
func WithSubscribeAutoAck(b bool) SubscribeOption {
	return func(o *SubscribeOptions) { o.AutoAck = b }
}

// WithSubscribeContext 覆盖订阅操作的上下文。
func WithSubscribeContext(ctx context.Context) SubscribeOption {
	return func(o *SubscribeOptions) { o.Context = ctx }
}

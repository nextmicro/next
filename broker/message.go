package broker

// Header 表示消息头部，key/value 均为字符串。
// 实现需支持 trace propagation、metadata 透传等横切关注点。
//
// Header 独立于 Payload 编码，使用底层 broker 的原生 header 机制
// （Kafka 的 record headers / memory broker 的内部 map）。
type Header map[string]string

// Get 返回 key 对应的值，不存在时返回空字符串。
func (h Header) Get(key string) string {
	if h == nil {
		return ""
	}
	return h[key]
}

// Set 设置 key 对应的值。若 Header 为 nil 则原样返回（请通过 Clone 或显式初始化）。
// 调用方应保证 Header 不为 nil。
func (h Header) Set(key, value string) {
	h[key] = value
}

// Clone 返回 Header 的浅拷贝。
func (h Header) Clone() Header {
	if h == nil {
		return nil
	}
	out := make(Header, len(h))
	for k, v := range h {
		out[k] = v
	}
	return out
}

// Message 是底层消息的字节级表示。
// 业务代码通常不直接构造 Message，而是通过 typed.Publisher[T] 间接发送。
type Message struct {
	// Header 是消息头部，可选。
	Header Header

	// Body 是消息载荷，已被序列化为字节流。
	Body []byte
}

package broker

// Wrapper 是 Broker 的装饰器函数。
// 接受一个 Broker，返回包装后的 Broker。
//
// 用于在不修改实现的前提下叠加 logging、metrics、tracing 等横切能力。
// 装饰器只在底层 []byte 层生效，对上层 typed.Publisher[T] / Subscriber[T] 透明，
// 不影响泛型层的编译期类型安全。
type Wrapper func(Broker) Broker

// Chain 将多个 Wrapper 组合为一个。
// 应用顺序为：Chain(a, b, c)(b) → a(b(c(b)))，即最先声明的最外层。
func Chain(ws ...Wrapper) Wrapper {
	return func(b Broker) Broker {
		for i := len(ws) - 1; i >= 0; i-- {
			b = ws[i](b)
		}
		return b
	}
}

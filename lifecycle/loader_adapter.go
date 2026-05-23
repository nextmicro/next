package lifecycle

import (
	"context"
	"reflect"
)

// FuncComponent 是一个通用 Component 构造器，
// 通过组合函数指针（任意可选）来声明 Init/Start/Stop/Watch/Health 行为。
//
// 适合：
//   - 把 next v1 的 Loader 适配为 Component（见 next 包顶层封装）
//   - 把第三方库（连接池、定时器、HTTP 客户端等）一次性挂到 Lifecycle
//   - 为单元测试快速构造期望行为的 Component
//
// 使用示例：
//
//	c := lifecycle.NewFuncComponent("metrics").
//	    WithStart(func(ctx context.Context) error { return server.ListenAndServe() }).
//	    WithStop(func(ctx context.Context) error { return server.Shutdown(ctx) })
//	lc.Add(c)
type FuncComponent struct {
	name   string
	init   func(context.Context) error
	start  func(context.Context) error
	stop   func(context.Context) error
	watch  func(context.Context) error
	health func(context.Context) error
	deps   []reflect.Type
}

// NewFuncComponent 创建一个新的可定制 Component。name 用于日志与依赖标识，必填。
func NewFuncComponent(name string) *FuncComponent {
	return &FuncComponent{name: name}
}

// Name 返回 Component 名称。
func (f *FuncComponent) Name() string { return f.name }

// WithInit 设置 Init 行为；不调用即不实现 Initializer。
func (f *FuncComponent) WithInit(fn func(context.Context) error) *FuncComponent {
	f.init = fn
	return f
}

// WithStart 设置 Start 行为；不调用即不实现 Starter。
func (f *FuncComponent) WithStart(fn func(context.Context) error) *FuncComponent {
	f.start = fn
	return f
}

// WithStop 设置 Stop 行为；不调用即不实现 Stopper。
func (f *FuncComponent) WithStop(fn func(context.Context) error) *FuncComponent {
	f.stop = fn
	return f
}

// WithWatch 设置 Watch 行为；不调用即不实现 Watcher。
func (f *FuncComponent) WithWatch(fn func(context.Context) error) *FuncComponent {
	f.watch = fn
	return f
}

// WithHealth 设置 Health 行为；不调用即不实现 HealthCheck。
func (f *FuncComponent) WithHealth(fn func(context.Context) error) *FuncComponent {
	f.health = fn
	return f
}

// WithDeps 声明依赖的其它 Component 类型，用于拓扑排序。
func (f *FuncComponent) WithDeps(deps ...reflect.Type) *FuncComponent {
	f.deps = append(f.deps, deps...)
	return f
}

// Init 调用注册的 init 函数；未设置时返回 nil（但仍实现 Initializer 接口）。
// 注意：FuncComponent 为简化使用静态实现所有可选接口，
// 编排器内部判断 nil 函数体跳过执行。
func (f *FuncComponent) Init(ctx context.Context) error {
	if f.init == nil {
		return nil
	}
	return f.init(ctx)
}

// Start 调用注册的 start 函数。
func (f *FuncComponent) Start(ctx context.Context) error {
	if f.start == nil {
		return nil
	}
	return f.start(ctx)
}

// Stop 调用注册的 stop 函数。
func (f *FuncComponent) Stop(ctx context.Context) error {
	if f.stop == nil {
		return nil
	}
	return f.stop(ctx)
}

// Watch 调用注册的 watch 函数。
func (f *FuncComponent) Watch(ctx context.Context) error {
	if f.watch == nil {
		return nil
	}
	return f.watch(ctx)
}

// Health 调用注册的 health 函数；未设置返回 nil（视作健康）。
func (f *FuncComponent) Health(ctx context.Context) error {
	if f.health == nil {
		return nil
	}
	return f.health(ctx)
}

// Deps 返回声明的依赖类型。
func (f *FuncComponent) Deps() []reflect.Type {
	if len(f.deps) == 0 {
		return nil
	}
	out := make([]reflect.Type, len(f.deps))
	copy(out, f.deps)
	return out
}

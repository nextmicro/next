// Package container 提供类型安全的依赖注入容器，使用 Go 泛型在编译期约束类型一致性。
//
// 设计目标：
//   - 消除 next 框架中的全局变量（DefaultBroker / DefaultRegistry / DefaultLogger 等）
//   - 启动期一次性注册，运行期热路径只读
//   - 使用 reflect.Type 作为键，与 Go 类型系统对齐
//   - 通过 Seal/Sealed 保证启动后不再变更
//
// 基本用法：
//
//	c := container.New()
//	_ = container.Register(c, &MyService{})
//	_ = container.RegisterAs[broker.Broker](c, &kafkaBroker{})
//	c.Seal()
//
//	svc, err := container.Resolve[*MyService](c)
//	bk := container.MustResolve[broker.Broker](c)
package container

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// ErrAlreadyRegistered 表示同一类型已经注册过。
var ErrAlreadyRegistered = errors.New("container: type already registered")

// ErrNotRegistered 表示请求解析的类型未注册。
var ErrNotRegistered = errors.New("container: type not registered")

// ErrSealed 表示容器已封印（Seal 后），不再接受写入。
var ErrSealed = errors.New("container: container is sealed")

// Container 是一个类型安全的依赖注入容器。
//
// 零值不可用，请使用 New() 构造。所有方法均为线程安全。
//
// 生命周期约定：
//  1. 启动期：调用 Register / RegisterAs / RegisterNamed 注册实例
//  2. 调用 Seal() 封印，禁止后续写入
//  3. 运行期：通过 Resolve / MustResolve / ResolveNamed 读取
type Container struct {
	mu     sync.RWMutex
	types  map[reflect.Type]any
	named  map[reflect.Type]map[string]any
	sealed bool
}

// New 创建一个新的空容器。
func New() *Container {
	return &Container{
		types: make(map[reflect.Type]any),
		named: make(map[reflect.Type]map[string]any),
	}
}

// Seal 封印容器。Seal 后任何写入操作（Register*）都将返回 ErrSealed。
// 重复 Seal 是幂等的。
func (c *Container) Seal() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sealed = true
}

// Sealed 返回容器是否已封印。
func (c *Container) Sealed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.sealed
}

// Register 以 T 的具体类型为键注册实例。
//
// 调用 Register[*MyService](c, svc) 后，可以通过 Resolve[*MyService](c) 取回。
// 重复注册同一类型返回 ErrAlreadyRegistered。
//
// 注意：T 是按 v 的静态类型推断的。若 v 是接口实现，要按接口类型注册，请使用 RegisterAs。
func Register[T any](c *Container, v T) error {
	t := typeOf[T]()
	return c.register(t, v)
}

// RegisterAs 以接口类型 I 为键注册实现。
//
// 用法：RegisterAs[broker.Broker](c, &kafkaBroker{})
// 之后 Resolve[broker.Broker](c) 将返回该实现。
func RegisterAs[I any](c *Container, v I) error {
	t := typeOf[I]()
	return c.register(t, v)
}

// RegisterNamed 以 (T, name) 为复合键注册命名实例。
//
// 用于同一类型多实例场景，例如多数据源、多 broker。
func RegisterNamed[T any](c *Container, name string, v T) error {
	if name == "" {
		return fmt.Errorf("container: name cannot be empty")
	}
	t := typeOf[T]()
	return c.registerNamed(t, name, v)
}

// Resolve 按 T 的类型解析实例。
//
// 若未注册，返回 T 的零值与 ErrNotRegistered 包装错误。
func Resolve[T any](c *Container) (T, error) {
	var zero T
	t := typeOf[T]()
	v, ok := c.lookup(t)
	if !ok {
		return zero, fmt.Errorf("%w: %s", ErrNotRegistered, t.String())
	}
	out, ok := v.(T)
	if !ok {
		return zero, fmt.Errorf("container: type assertion failed for %s", t.String())
	}
	return out, nil
}

// MustResolve 等同 Resolve，但在失败时 panic。
//
// 仅用于初始化阶段或调用方明确知道类型已注册的场景。
func MustResolve[T any](c *Container) T {
	v, err := Resolve[T](c)
	if err != nil {
		panic(err)
	}
	return v
}

// ResolveNamed 按 (T, name) 解析命名实例。
func ResolveNamed[T any](c *Container, name string) (T, error) {
	var zero T
	t := typeOf[T]()
	v, ok := c.lookupNamed(t, name)
	if !ok {
		return zero, fmt.Errorf("%w: %s (name=%q)", ErrNotRegistered, t.String(), name)
	}
	out, ok := v.(T)
	if !ok {
		return zero, fmt.Errorf("container: type assertion failed for %s (name=%q)", t.String(), name)
	}
	return out, nil
}

// Validate 校验 deps 中所有依赖均可解析。
//
// deps 的键为引用方标识（通常是 Component 名称），值为该方依赖的类型列表。
// 返回的错误聚合所有缺失项，便于一次性报错。
func (c *Container) Validate(deps map[string][]reflect.Type) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var errs []error
	for owner, requires := range deps {
		for _, t := range requires {
			if _, ok := c.types[t]; ok {
				continue
			}
			errs = append(errs, fmt.Errorf("container: %s depends on unregistered type %s", owner, t.String()))
		}
	}
	return errors.Join(errs...)
}

// Has 返回 T 是否已注册。
func Has[T any](c *Container) bool {
	t := typeOf[T]()
	_, ok := c.lookup(t)
	return ok
}

// HasNamed 返回 (T, name) 是否已注册。
func HasNamed[T any](c *Container, name string) bool {
	t := typeOf[T]()
	_, ok := c.lookupNamed(t, name)
	return ok
}

// register 是 Register / RegisterAs 的内部实现。
func (c *Container) register(t reflect.Type, v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sealed {
		return fmt.Errorf("%w: cannot register %s", ErrSealed, t.String())
	}
	if _, exists := c.types[t]; exists {
		return fmt.Errorf("%w: %s", ErrAlreadyRegistered, t.String())
	}
	c.types[t] = v
	return nil
}

// registerNamed 是 RegisterNamed 的内部实现。
func (c *Container) registerNamed(t reflect.Type, name string, v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sealed {
		return fmt.Errorf("%w: cannot register %s (name=%q)", ErrSealed, t.String(), name)
	}
	bucket, ok := c.named[t]
	if !ok {
		bucket = make(map[string]any)
		c.named[t] = bucket
	}
	if _, exists := bucket[name]; exists {
		return fmt.Errorf("%w: %s (name=%q)", ErrAlreadyRegistered, t.String(), name)
	}
	bucket[name] = v
	return nil
}

func (c *Container) lookup(t reflect.Type) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.types[t]
	return v, ok
}

func (c *Container) lookupNamed(t reflect.Type, name string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	bucket, ok := c.named[t]
	if !ok {
		return nil, false
	}
	v, ok := bucket[name]
	return v, ok
}

// typeOf 返回 T 的 reflect.Type。
// 当 T 是接口类型时，直接返回该接口类型而非具体实现类型。
func typeOf[T any]() reflect.Type {
	return reflect.TypeOf((*T)(nil)).Elem()
}

// TypeOf 是 typeOf 的导出版本，便于 Component 在 Deps() 中使用：
//
//	func (b *BrokerComponent) Deps() []reflect.Type {
//	    return []reflect.Type{container.TypeOf[*LoggerComponent]()}
//	}
func TypeOf[T any]() reflect.Type {
	return typeOf[T]()
}

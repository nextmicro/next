// Package lifecycle 提供基于拓扑排序的 Component 生命周期编排。
//
// 每个 Component 通过接口组合按需声明能力：
//   - Initializer：启动前的初始化（读配置、建立懒连接等）
//   - Starter：实际启动（建立连接、启动后台 goroutine 等）
//   - Stopper：停止与资源释放
//   - Watcher：注册配置/事件监听
//   - HealthCheck：健康检查
//   - DependsOn：声明对其他 Component 类型的依赖
//
// 编排约定：
//   - Start 时按依赖拓扑排序、分层、同层并行启动
//   - 任意 Start 失败时反向回滚已启动 Component
//   - Stop 按相反顺序执行，每个 Component 受 context 超时控制
//   - Health 聚合所有 HealthCheck Component 的结果
package lifecycle

import (
	"context"
	"reflect"
)

// Component 是所有生命周期组件的根接口。
//
// Component 必须有可读的名称（用于日志、错误信息、健康检查报告）。
// 能力（Init/Start/Stop/Watch/Health/Deps）通过额外接口按需实现。
type Component interface {
	Name() string
}

// Initializer 是可选的初始化能力。
// Init 在 Lifecycle.Start 之前由编排器调用，可访问 Container 解析依赖。
// 传入的 ctx 用于级联取消；Init 不应阻塞。
type Initializer interface {
	Init(ctx context.Context) error
}

// Starter 是可选的启动能力。
// Start 必须在依赖 Component 全部 Start 成功后才会被调用。
// 实现应在 ctx 超时或取消时尽快返回，但允许保留后台 goroutine 在 Stop 时清理。
type Starter interface {
	Start(ctx context.Context) error
}

// Stopper 是可选的停止能力。
// Stop 必须释放该 Component 占用的所有资源（连接、goroutine、文件句柄）。
// 实现应尊重 ctx 超时；超时未完成时编排器会强制继续下一个。
type Stopper interface {
	Stop(ctx context.Context) error
}

// Watcher 是可选的监听能力。
// Watch 在所有 Component Start 成功后被并发调用，用于注册配置变更回调等。
// Watch 不应阻塞当前 goroutine。
type Watcher interface {
	Watch(ctx context.Context) error
}

// HealthCheck 是可选的健康检查能力。
// Lifecycle.Health 会并发调用所有实现该接口的 Component。
type HealthCheck interface {
	Health(ctx context.Context) error
}

// DependsOn 是可选的依赖声明能力。
// 返回该 Component 依赖的其它 Component 的类型（reflect.Type）。
// 编排器据此进行拓扑排序与 Container Validate。
//
// 示例：
//
//	func (b *BrokerComponent) Deps() []reflect.Type {
//	    return []reflect.Type{container.TypeOf[*LoggerComponent]()}
//	}
type DependsOn interface {
	Deps() []reflect.Type
}

// componentType 返回 c 的具体类型（指针类型）。
// 用于将 Component 实例映射到 reflect.Type 键。
func componentType(c Component) reflect.Type {
	return reflect.TypeOf(c)
}

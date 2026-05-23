// Package components 提供 next 框架开箱即用的 lifecycle.Component 集合。
//
// 设计思路：保持向后兼容的同时把 next v1 的 4 个 runtime loader
// （logger / tracing / registry / broker）暴露为新 lifecycle 体系的 Component。
//
// 目前内部依然委托给 `runtime/loader/{...}` 实现，
// 长远计划是逐个改写为独立 Component（不依赖旧 Loader 接口），届时本包是稳定入口。
//
// 业务方典型用法：
//
//	app, err := next.New(
//	    next.WithComponents(
//	        components.DefaultLogger(),
//	        components.DefaultTracing(),
//	        components.DefaultRegistry(),
//	        components.DefaultBroker(),
//	        myCustom,
//	    ),
//	)
package components

import (
	"context"

	"github.com/nextmicro/next/lifecycle"
	"github.com/nextmicro/next/runtime/loader"
	brokerldr "github.com/nextmicro/next/runtime/loader/broker"
	loggerldr "github.com/nextmicro/next/runtime/loader/logger"
	registryldr "github.com/nextmicro/next/runtime/loader/registry"
	tracingldr "github.com/nextmicro/next/runtime/loader/tracing"
)

// wrapLoader 把 next v1 Loader 包装为 lifecycle.Component。
//
// 包装语义：
//   - Init 直接调用 Loader.Init()；不论是否 Initialized，错误均向上传播
//   - Start 仅当 Initialized 返回 true 时调用 Loader.Start(ctx)
//   - Stop 同上；上层编排器超时控制
//   - Watch 同上；旧接口不接受 ctx
func wrapLoader(name string, ldr loader.Loader) lifecycle.Component {
	return lifecycle.NewFuncComponent(name).
		WithInit(func(_ context.Context) error {
			return ldr.Init()
		}).
		WithStart(func(ctx context.Context) error {
			if !ldr.Initialized() {
				return nil
			}
			return ldr.Start(ctx)
		}).
		WithStop(func(ctx context.Context) error {
			if !ldr.Initialized() {
				return nil
			}
			return ldr.Stop(ctx)
		}).
		WithWatch(func(_ context.Context) error {
			if !ldr.Initialized() {
				return nil
			}
			return ldr.Watch()
		})
}

// DefaultLogger 返回 next 默认 Logger Component。
// 读取 ApplicationConfig().Logger 并初始化 nextmicro/logger，同时设置 kratos / nacos 的 logger 适配器。
func DefaultLogger() lifecycle.Component {
	return wrapLoader("logger", loggerldr.New())
}

// DefaultTracing 返回 next 默认 OpenTelemetry Tracing Component。
// 读取 ApplicationConfig().Telemetry；如 disable 则 Init 返回 nil 且 Initialized 为 false。
func DefaultTracing() lifecycle.Component {
	return wrapLoader("tracing", tracingldr.New())
}

// DefaultRegistry 返回 next 默认服务注册 Component。
// 读取 ApplicationConfig().Registry，按 name 选择 consul / etcd / memory。
func DefaultRegistry() lifecycle.Component {
	return wrapLoader("registry", registryldr.New())
}

// DefaultBroker 返回 next 默认 Broker Component。
// 读取 ApplicationConfig().Broker，按 name 选择 kafka 或 memory。
func DefaultBroker() lifecycle.Component {
	return wrapLoader("broker", brokerldr.New())
}

// All 返回 next 默认提供的全部 Component（按 Add 顺序）。
// 等价于：[DefaultLogger, DefaultTracing, DefaultRegistry, DefaultBroker]
func All() []lifecycle.Component {
	return []lifecycle.Component{
		DefaultLogger(),
		DefaultTracing(),
		DefaultRegistry(),
		DefaultBroker(),
	}
}

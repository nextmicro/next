// Package next 是基于 go-kratos 的微服务装饰扩展层。
//
// next v2 的核心改动（相对 v1）：
//   - 引入 container.Container：基于 generics 的类型安全 DI，消除全局变量
//   - 引入 lifecycle.Lifecycle：基于拓扑排序的 Component 编排器
//   - 中间件注册改为可实例化的 middleware.Registry
//   - Broker 抽象增加 broker/typed 子包：类型安全的 Publisher[T]/Subscriber[T]
//
// 入口：
//
//	app, err := next.New(next.ID("..."), next.Name("..."), ...)
//	if err != nil { panic(err) }
//	if err := app.Run(); err != nil { panic(err) }
package next

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v2"
	kconfig "github.com/go-kratos/kratos/v2/config"
	configv1 "github.com/nextmicro/next/api/config/v1"
	"github.com/nextmicro/next/components"
	"github.com/nextmicro/next/config"
	"github.com/nextmicro/next/container"
	"github.com/nextmicro/next/lifecycle"
	"github.com/nextmicro/next/middleware"
	"github.com/nextmicro/next/registry"
)

// 注：v2 不再 blank-import 所有 middleware 子包。业务方按需选择：
//
//	import _ "github.com/nextmicro/next/middleware/all"          // 全部
//	import _ "github.com/nextmicro/next/middleware/tracing"      // 仅 tracing
//
// 见 middleware/all/all.go 注释。

// Next 是 v1 名称别名，保留为类型别名以兼容业务代码 `var app *next.Next`。
//
// Deprecated: 直接使用 *next.App。
type Next = App

// New 构造一个 next v2 App：
//  1. 从 config.ApplicationConfig() 读取默认 id/name/version/metadata
//  2. 合并业务传入的 Option
//  3. 创建 Container 与 Lifecycle，添加默认 4 个基础设施 Component 与业务 Component
//  4. 执行 WithContainer 注册函数，让业务往 Container 注入服务
//  5. 注册 middleware.Registry 到 Container
//  6. 创建 kratos.App（嵌入 BeforeStart/AfterStop 等 Option）
//  7. 写入构建信息 metrics
//
// 注意：本函数不会真正启动 Lifecycle；启动延迟到 app.Run()。
func New(opts ...Option) (*App, error) {
	opt := buildAppOptions(opts...)

	c := container.New()
	if err := container.Register(c, c); err != nil {
		return nil, fmt.Errorf("container self-register: %w", err)
	}

	mwReg := opt.mwRegistry
	if mwReg == nil {
		mwReg = middleware.DefaultRegistry
	}
	if err := container.RegisterAs[middleware.Registry](c, mwReg); err != nil {
		return nil, fmt.Errorf("container register middleware.Registry: %w", err)
	}
	if appCfg := config.ApplicationConfig(); appCfg != nil {
		if err := container.Register[*configv1.Next](c, appCfg); err != nil {
			return nil, fmt.Errorf("container register *configv1.Next: %w", err)
		}
	}
	if kc := config.DefaultConfig; kc != nil {
		if err := container.RegisterAs[kconfig.Config](c, kc); err != nil {
			return nil, fmt.Errorf("container register kconfig.Config: %w", err)
		}
	}

	lc := lifecycle.New()

	for _, comp := range defaultComponents(opt) {
		lc.Add(comp)
	}
	for _, comp := range opt.components {
		lc.Add(comp)
	}
	for _, ldr := range opt.legacyLoaders {
		ldr := ldr
		lc.Add(lifecycle.NewFuncComponent(ldr.String()).
			WithInit(func(_ context.Context) error { return ldr.Init() }).
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
			}))
	}

	for _, fn := range opt.containerFn {
		fn(c)
	}

	kOpts := buildKratosOptions(opt)
	kApp := kratos.New(kOpts...)

	app := &App{
		App:        kApp,
		opt:        opt,
		container:  c,
		lifecycle:  lc,
		mwRegistry: mwReg,
	}
	app.initBuildMetrics()
	return app, nil
}

// buildAppOptions 合并配置文件中的默认值与业务传入的 Option。
//
// 优先级（高 → 低）：业务 Option > 配置文件 > 内置默认。
func buildAppOptions(in ...Option) options {
	opt := defaultOptions()

	// 配置文件中的默认值（仅在业务未显式覆盖时生效）
	cfg := config.ApplicationConfig()
	defaultsFromCfg := make([]Option, 0, 4)
	if v := cfg.GetId(); v != "" {
		defaultsFromCfg = append(defaultsFromCfg, ID(v))
	}
	if v := cfg.GetName(); v != "" {
		defaultsFromCfg = append(defaultsFromCfg, Name(v))
	}
	if v := cfg.GetVersion(); v != "" {
		defaultsFromCfg = append(defaultsFromCfg, Version(v))
	}
	if md := cfg.GetMetadata(); md != nil {
		defaultsFromCfg = append(defaultsFromCfg, Metadata(md))
	}

	for _, o := range defaultsFromCfg {
		o(&opt)
	}
	for _, o := range in {
		o(&opt)
	}

	if opt.registrar == nil {
		opt.registrar = registry.DefaultRegistry
	}
	return opt
}

// defaultComponents 当业务未通过 WithComponents 传入任何 Component 时，
// 返回 next 默认提供的基础设施 Component（logger/tracing/registry/broker）。
//
// 若业务传入了 WithComponents，则视为业务自行管理，不再追加默认 Component
// （避免重复注册 logger 等导致冲突）。
func defaultComponents(opt options) []lifecycle.Component {
	if len(opt.components) > 0 || len(opt.legacyLoaders) > 0 {
		return nil
	}
	return components.All()
}

// buildKratosOptions 把 next.options 翻译为 kratos.Option 列表。
func buildKratosOptions(opt options) []kratos.Option {
	kOpts := []kratos.Option{
		kratos.ID(opt.id),
		kratos.Name(opt.name),
		kratos.Version(opt.version),
		kratos.Metadata(opt.metadata),
		kratos.Endpoint(opt.endpoints...),
		kratos.Context(opt.ctx),
		kratos.Server(opt.servers...),
		kratos.Signal(opt.sigs...),
		kratos.Registrar(opt.registrar),
		kratos.RegistrarTimeout(opt.registrarTimeout),
		kratos.StopTimeout(opt.stopTimeout),
	}
	for _, fn := range opt.beforeStart {
		kOpts = append(kOpts, kratos.BeforeStart(fn))
	}
	for _, fn := range opt.beforeStop {
		kOpts = append(kOpts, kratos.BeforeStop(fn))
	}
	for _, fn := range opt.afterStart {
		kOpts = append(kOpts, kratos.AfterStart(fn))
	}
	for _, fn := range opt.afterStop {
		kOpts = append(kOpts, kratos.AfterStop(fn))
	}
	return kOpts
}

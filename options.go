package next

import (
	"context"
	"net/url"
	"os"
	"syscall"
	"time"

	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/nextmicro/logger"
	"github.com/nextmicro/next/container"
	"github.com/nextmicro/next/lifecycle"
	"github.com/nextmicro/next/middleware"
	"github.com/nextmicro/next/runtime/loader"
)

// options 是新 App 的内部配置载体。
// 不导出，业务方通过 Option 函数组合配置。
type options struct {
	id        string
	name      string
	version   string
	metadata  map[string]string
	endpoints []*url.URL

	ctx  context.Context
	sigs []os.Signal

	logger           logger.Logger
	registrar        registry.Registrar
	registrarTimeout time.Duration
	stopTimeout      time.Duration

	servers     []transport.Server
	components  []lifecycle.Component
	mwRegistry  middleware.Registry
	containerFn []func(*container.Container)

	beforeStart []func(context.Context) error
	beforeStop  []func(context.Context) error
	afterStart  []func(context.Context) error
	afterStop   []func(context.Context) error

	// legacy: 兼容 v1 的 Loader（仍允许传入；New 会自动包装为 Component）
	legacyLoaders []loader.Loader
}

// Option 是 App 配置函数。
type Option func(*options)

// defaultOptions 返回新 App 的默认配置。
func defaultOptions() options {
	return options{
		ctx:              context.Background(),
		sigs:             []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT},
		stopTimeout:      10 * time.Second,
		registrarTimeout: 10 * time.Second,
	}
}

// ============================================================
// 基础元数据
// ============================================================

// ID 设置服务 ID。
func ID(id string) Option { return func(o *options) { o.id = id } }

// Name 设置服务名。
func Name(name string) Option { return func(o *options) { o.name = name } }

// Version 设置服务版本。
func Version(version string) Option { return func(o *options) { o.version = version } }

// Metadata 设置服务元数据。
func Metadata(md map[string]string) Option { return func(o *options) { o.metadata = md } }

// Endpoint 设置对外暴露的 endpoint URL 列表。
func Endpoint(endpoints ...*url.URL) Option {
	return func(o *options) { o.endpoints = endpoints }
}

// Context 设置启动用 context；默认 context.Background()。
func Context(ctx context.Context) Option {
	return func(o *options) { o.ctx = ctx }
}

// Signal 设置退出信号集合；默认 SIGTERM/SIGINT/SIGQUIT。
func Signal(sigs ...os.Signal) Option {
	return func(o *options) { o.sigs = sigs }
}

// ============================================================
// 基础设施
// ============================================================

// Logger 设置应用 Logger（兼容 v1 Option，但新代码建议通过 Component 注入）。
func Logger(l logger.Logger) Option {
	return func(o *options) { o.logger = l }
}

// Server 设置 kratos transport.Server 列表。
func Server(srv ...transport.Server) Option {
	return func(o *options) { o.servers = srv }
}

// Registrar 设置服务注册器（kratos.Registrar）。
func Registrar(r registry.Registrar) Option {
	return func(o *options) { o.registrar = r }
}

// RegistrarTimeout 设置 Registrar 操作超时。
func RegistrarTimeout(t time.Duration) Option {
	return func(o *options) { o.registrarTimeout = t }
}

// StopTimeout 设置 Stop 超时。
func StopTimeout(t time.Duration) Option {
	return func(o *options) { o.stopTimeout = t }
}

// ============================================================
// v2 新增：Container & Components
// ============================================================

// WithComponents 追加业务自定义 lifecycle.Component。
// 多次调用按顺序累加；编排器按拓扑分层启动。
func WithComponents(comps ...lifecycle.Component) Option {
	return func(o *options) { o.components = append(o.components, comps...) }
}

// WithContainer 注册一个在 App 构造期对 Container 进行自定义注册的函数。
// 多次调用按顺序执行。函数应只做 Register；不应在此处启动 goroutine 或建立连接。
//
// 示例：
//
//	next.WithContainer(func(c *container.Container) {
//	    container.Register(c, mybiz.NewService())
//	    container.RegisterAs[contracts.UserRepo](c, sqlrepo.New())
//	})
func WithContainer(fn func(*container.Container)) Option {
	return func(o *options) { o.containerFn = append(o.containerFn, fn) }
}

// WithMiddlewareRegistry 注入自定义 middleware.Registry。
// 默认 App 会创建一个全新 Registry（不复用 middleware.DefaultRegistry）。
// 业务方如希望复用兼容期 DefaultRegistry，可显式传入。
func WithMiddlewareRegistry(r middleware.Registry) Option {
	return func(o *options) { o.mwRegistry = r }
}

// ============================================================
// v1 兼容：Loader Option
// ============================================================

// Loader 追加 v1 风格 Loader。
//
// Deprecated: 仅为兼容已有业务代码暂时保留。新代码请使用 WithComponents 直接传入 Component。
// App 构造时会通过 lifecycle.NewFuncComponent 将每个 Loader 包装为 Component。
func Loader(loaders ...loader.Loader) Option {
	return func(o *options) { o.legacyLoaders = append(o.legacyLoaders, loaders...) }
}

// ============================================================
// Lifecycle Hooks（兼容 kratos）
// ============================================================

// BeforeStart 注册启动前钩子（透传给 kratos.BeforeStart）。
func BeforeStart(fn func(context.Context) error) Option {
	return func(o *options) { o.beforeStart = append(o.beforeStart, fn) }
}

// BeforeStop 注册停止前钩子（透传给 kratos.BeforeStop）。
func BeforeStop(fn func(context.Context) error) Option {
	return func(o *options) { o.beforeStop = append(o.beforeStop, fn) }
}

// AfterStart 注册启动后钩子（透传给 kratos.AfterStart）。
func AfterStart(fn func(context.Context) error) Option {
	return func(o *options) { o.afterStart = append(o.afterStart, fn) }
}

// AfterStop 注册停止后钩子（透传给 kratos.AfterStop）。
func AfterStop(fn func(context.Context) error) Option {
	return func(o *options) { o.afterStop = append(o.afterStop, fn) }
}

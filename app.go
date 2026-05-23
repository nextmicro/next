package next

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2"
	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	"github.com/nextmicro/next/container"
	"github.com/nextmicro/next/lifecycle"
	"github.com/nextmicro/next/middleware"
	"github.com/nextmicro/next/pkg/env"
	metric "github.com/nextmicro/next/pkg/metrics"
)

// App 是 next v2 的应用容器，装饰 kratos.App，集成 Container 与 Lifecycle 编排。
//
// 与 v1 Next 结构（嵌入 *kratos.App + Options）相比：
//   - 持有 *container.Container：业务可通过它注册/解析服务，避免全局变量
//   - 持有 *lifecycle.Lifecycle：编排 logger/tracing/broker/registry 等基础设施的启停
//   - Run/Stop 是组合：先 Lifecycle.Start → 再 kratos.App.Run；Stop 时反向执行
//
// 用法：
//
//	app, err := next.New(
//	    next.ID("svc-1"),
//	    next.Name("orders"),
//	    next.WithComponents(components.All()...),
//	    next.WithContainer(func(c *container.Container) {
//	        container.Register(c, mybiz.NewService())
//	    }),
//	)
//	if err != nil { ... }
//	defer app.Stop(context.Background())
//	if err := app.Run(); err != nil { ... }
type App struct {
	*kratos.App

	opt        options
	container  *container.Container
	lifecycle  *lifecycle.Lifecycle
	mwRegistry middleware.Registry
}

// Container 返回应用持有的 DI 容器。
func (a *App) Container() *container.Container { return a.container }

// Lifecycle 返回应用持有的 Component 编排器。
func (a *App) Lifecycle() *lifecycle.Lifecycle { return a.lifecycle }

// MiddlewareRegistry 返回应用持有的 middleware 注册表。
func (a *App) MiddlewareRegistry() middleware.Registry { return a.mwRegistry }

// Run 启动应用：
//  1. 依次执行 BeforeStart 钩子
//  2. Lifecycle.Start（按拓扑分层并行启动 Component）
//  3. 委托 kratos.App.Run（阻塞直到收到信号或所有 Server.Stop 完成）
//  4. 退出后执行 Lifecycle.Stop（反向停止 Component），返回首要错误
//
// 注意：BeforeStop / AfterStop / AfterStart 钩子已通过 kratos.Option 在 New() 中注册，
// 此处不重复执行。
func (a *App) Run() error {
	startCtx := a.opt.ctx
	if startCtx == nil {
		startCtx = context.Background()
	}

	if err := a.lifecycle.Start(startCtx); err != nil {
		return fmt.Errorf("lifecycle start: %w", err)
	}

	runErr := a.App.Run()

	stopCtx, cancel := context.WithTimeout(context.Background(), a.opt.stopTimeout)
	defer cancel()
	stopErr := a.lifecycle.Stop(stopCtx)

	switch {
	case runErr != nil && stopErr != nil:
		return errors.Join(runErr, stopErr)
	case runErr != nil:
		return runErr
	default:
		return stopErr
	}
}

// initBuildMetrics 写入构建信息到 prometheus（保持与 v1 一致）。
func (a *App) initBuildMetrics() {
	prom.NewGauge(metric.BuildInfoGauge).With(
		a.ID(),
		a.Name(),
		a.Version(),
		env.DeployEnvironment(),
		env.GoVersion(),
		env.AppVersion(),
		env.StartTime(),
		env.BuildTime(),
	).Set(float64(time.Now().UnixNano() / 1e6))
}

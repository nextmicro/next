package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// HealthStatus 表示整体健康状态。
type HealthStatus int

const (
	// HealthOK 所有 HealthCheck Component 都返回 nil。
	HealthOK HealthStatus = iota
	// HealthDegraded 至少一个 HealthCheck 返回错误。
	HealthDegraded
)

func (s HealthStatus) String() string {
	switch s {
	case HealthOK:
		return "OK"
	case HealthDegraded:
		return "DEGRADED"
	default:
		return "UNKNOWN"
	}
}

// HealthReport 是 Health(ctx) 的聚合结果。
type HealthReport struct {
	Status  HealthStatus
	Details map[string]error // key: Component.Name(), value: 错误（nil 表示健康）
}

// Lifecycle 是 Component 的生命周期编排器。
//
// 用法：
//
//	lc := lifecycle.New()
//	lc.Add(logger).Add(tracing).Add(broker).Add(server)
//	if err := lc.Start(ctx); err != nil { ... }
//	defer lc.Stop(ctx)
type Lifecycle struct {
	mu sync.Mutex

	comps   []Component
	started []Component // 反向 Stop 用，按启动顺序追加

	// 单个 Component 的 Stop 超时；0 表示使用 ctx 自身超时
	stopTimeoutPerComponent time.Duration
}

// New 创建一个新的 Lifecycle 编排器。
func New(opts ...Option) *Lifecycle {
	lc := &Lifecycle{
		stopTimeoutPerComponent: 30 * time.Second,
	}
	for _, o := range opts {
		o(lc)
	}
	return lc
}

// Option 配置 Lifecycle 行为。
type Option func(*Lifecycle)

// WithStopTimeoutPerComponent 设置单个 Component Stop 的超时。
// 默认 30 秒。
func WithStopTimeoutPerComponent(d time.Duration) Option {
	return func(l *Lifecycle) { l.stopTimeoutPerComponent = d }
}

// Add 添加一个 Component 到编排器。
// Add 必须在 Start 之前调用，否则返回未启动的 Component 是未定义行为。
func (l *Lifecycle) Add(c Component) *Lifecycle {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.comps = append(l.comps, c)
	return l
}

// Components 返回所有已注册 Component 的快照（用于 Container.Validate）。
func (l *Lifecycle) Components() []Component {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Component, len(l.comps))
	copy(out, l.comps)
	return out
}

// CollectDeps 返回 Component.Name → 依赖类型的映射，供 Container.Validate 使用。
func (l *Lifecycle) CollectDeps() map[string][]reflect.Type {
	return collectDeps(l.Components())
}

// Start 启动所有 Component：
//  1. 按依赖拓扑排序并分层
//  2. 逐层执行 Init → Start，同层并行
//  3. 同层任一 Component 失败时聚合错误，反向回滚已成功 Component
//  4. 全部成功后并发调用 Watcher.Watch
//
// 启动后 Component 进入"已启动"状态，后续 Stop 按反向顺序释放。
func (l *Lifecycle) Start(ctx context.Context) error {
	l.mu.Lock()
	comps := append([]Component(nil), l.comps...)
	l.mu.Unlock()

	layers, err := topoSort(comps)
	if err != nil {
		return err
	}

	for _, layer := range layers {
		if err := l.startLayer(ctx, layer); err != nil {
			// 启动失败 → 反向回滚已启动的 Component
			rollbackErr := l.rollback(ctx)
			if rollbackErr != nil {
				return errors.Join(err, fmt.Errorf("rollback: %w", rollbackErr))
			}
			return err
		}
	}

	// 全部启动后注册 Watcher
	if err := l.watchAll(ctx); err != nil {
		return err
	}

	return nil
}

// startLayer 并行启动一层 Component。
// 任一失败立即取消其余，聚合所有错误返回。
func (l *Lifecycle) startLayer(ctx context.Context, layer []Component) error {
	eg, gctx := errgroup.WithContext(ctx)

	// 用于记录本层成功启动的 Component
	var layerMu sync.Mutex
	layerStarted := make([]Component, 0, len(layer))

	for _, c := range layer {
		c := c
		eg.Go(func() error {
			if init, ok := c.(Initializer); ok {
				if err := init.Init(gctx); err != nil {
					return fmt.Errorf("init %s: %w", c.Name(), err)
				}
			}
			if starter, ok := c.(Starter); ok {
				if err := starter.Start(gctx); err != nil {
					return fmt.Errorf("start %s: %w", c.Name(), err)
				}
			}
			layerMu.Lock()
			layerStarted = append(layerStarted, c)
			layerMu.Unlock()
			return nil
		})
	}

	err := eg.Wait()

	// 将本层成功启动的 Component 追加到 l.started（按完成顺序，因并行不保证启动顺序）
	l.mu.Lock()
	l.started = append(l.started, layerStarted...)
	l.mu.Unlock()

	return err
}

// watchAll 并发调用所有 Watcher.Watch。
// Watch 本身不应阻塞；若多个 Watcher 同时返回错误，聚合返回。
func (l *Lifecycle) watchAll(ctx context.Context) error {
	l.mu.Lock()
	comps := append([]Component(nil), l.comps...)
	l.mu.Unlock()

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	for _, c := range comps {
		w, ok := c.(Watcher)
		if !ok {
			continue
		}
		wg.Add(1)
		go func(name string, w Watcher) {
			defer wg.Done()
			if err := w.Watch(ctx); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("watch %s: %w", name, err))
				mu.Unlock()
			}
		}(c.Name(), w)
	}
	wg.Wait()
	return errors.Join(errs...)
}

// rollback 反向停止已启动的 Component（启动失败回滚时调用）。
func (l *Lifecycle) rollback(ctx context.Context) error {
	l.mu.Lock()
	started := l.started
	l.started = nil
	l.mu.Unlock()

	return l.stopComponents(ctx, started)
}

// Stop 反向停止所有 Component。
// 每个 Component 受 stopTimeoutPerComponent 控制；超时未完成则记录错误并继续下一个。
// 所有错误聚合返回。
func (l *Lifecycle) Stop(ctx context.Context) error {
	l.mu.Lock()
	started := l.started
	l.started = nil
	l.mu.Unlock()

	return l.stopComponents(ctx, started)
}

func (l *Lifecycle) stopComponents(ctx context.Context, started []Component) error {
	var errs []error
	for i := len(started) - 1; i >= 0; i-- {
		c := started[i]
		stopper, ok := c.(Stopper)
		if !ok {
			continue
		}

		stopCtx, cancel := l.deriveStopCtx(ctx)
		err := stopper.Stop(stopCtx)
		cancel()
		if err != nil && !errors.Is(err, context.Canceled) {
			errs = append(errs, fmt.Errorf("stop %s: %w", c.Name(), err))
		}
	}
	return errors.Join(errs...)
}

func (l *Lifecycle) deriveStopCtx(parent context.Context) (context.Context, context.CancelFunc) {
	if l.stopTimeoutPerComponent <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, l.stopTimeoutPerComponent)
}

// Health 并发收集所有 HealthCheck Component 的状态并聚合。
func (l *Lifecycle) Health(ctx context.Context) HealthReport {
	l.mu.Lock()
	comps := append([]Component(nil), l.comps...)
	l.mu.Unlock()

	report := HealthReport{
		Status:  HealthOK,
		Details: make(map[string]error),
	}

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)
	for _, c := range comps {
		hc, ok := c.(HealthCheck)
		if !ok {
			continue
		}
		wg.Add(1)
		go func(name string, hc HealthCheck) {
			defer wg.Done()
			err := hc.Health(ctx)
			mu.Lock()
			report.Details[name] = err
			if err != nil {
				report.Status = HealthDegraded
			}
			mu.Unlock()
		}(c.Name(), hc)
	}
	wg.Wait()
	return report
}

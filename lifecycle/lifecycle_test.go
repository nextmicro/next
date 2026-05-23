package lifecycle_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nextmicro/next/container"
	"github.com/nextmicro/next/lifecycle"
)

// fakeComp 是一个可配置的测试 Component。
// 通过具体类型而非接口注册，便于 reflect.TypeOf 索引。
type fakeComp struct {
	name      string
	deps      []reflect.Type
	startErr  error
	stopErr   error
	healthErr error

	startDelay time.Duration
	stopDelay  time.Duration

	started atomic.Bool
	stopped atomic.Bool

	startedAt time.Time
	stoppedAt time.Time

	startCount atomic.Int32
	stopCount  atomic.Int32
}

func (f *fakeComp) Name() string             { return f.name }
func (f *fakeComp) Deps() []reflect.Type     { return f.deps }
func (f *fakeComp) Health(_ context.Context) error { return f.healthErr }

func (f *fakeComp) Start(ctx context.Context) error {
	f.startCount.Add(1)
	if f.startDelay > 0 {
		select {
		case <-time.After(f.startDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if f.startErr != nil {
		return f.startErr
	}
	f.started.Store(true)
	f.startedAt = time.Now()
	return nil
}

func (f *fakeComp) Stop(ctx context.Context) error {
	f.stopCount.Add(1)
	if f.stopDelay > 0 {
		select {
		case <-time.After(f.stopDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	f.stopped.Store(true)
	f.stoppedAt = time.Now()
	return f.stopErr
}

// 用于"分组测试"，每个测试用一组独立类型避免拓扑冲突
type compA struct{ fakeComp }
type compB struct{ fakeComp }
type compC struct{ fakeComp }

func newA(name string, deps ...reflect.Type) *compA {
	return &compA{fakeComp: fakeComp{name: name, deps: deps}}
}
func newB(name string, deps ...reflect.Type) *compB {
	return &compB{fakeComp: fakeComp{name: name, deps: deps}}
}
func newC(name string, deps ...reflect.Type) *compC {
	return &compC{fakeComp: fakeComp{name: name, deps: deps}}
}

func TestStart_StopReverseOrder(t *testing.T) {
	a := newA("a")
	b := newB("b", container.TypeOf[*compA]())
	c := newC("c", container.TypeOf[*compB]())

	lc := lifecycle.New()
	lc.Add(a).Add(b).Add(c)

	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !a.started.Load() || !b.started.Load() || !c.started.Load() {
		t.Fatalf("not all started")
	}
	if !a.startedAt.Before(b.startedAt) {
		t.Fatalf("a should start before b")
	}
	if !b.startedAt.Before(c.startedAt) {
		t.Fatalf("b should start before c")
	}

	if err := lc.Stop(context.Background()); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
	if !a.stopped.Load() || !b.stopped.Load() || !c.stopped.Load() {
		t.Fatalf("not all stopped")
	}
	if !c.stoppedAt.Before(b.stoppedAt) {
		t.Fatalf("c should stop before b (reverse)")
	}
	if !b.stoppedAt.Before(a.stoppedAt) {
		t.Fatalf("b should stop before a (reverse)")
	}
}

func TestStart_ParallelSameLayer(t *testing.T) {
	// a 是根，b/c 都依赖 a 但彼此无依赖 → 同层并行
	a := newA("a")
	b := newB("b", container.TypeOf[*compA]())
	c := newC("c", container.TypeOf[*compA]())

	b.startDelay = 100 * time.Millisecond
	c.startDelay = 100 * time.Millisecond

	lc := lifecycle.New()
	lc.Add(a).Add(b).Add(c)

	start := time.Now()
	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	elapsed := time.Since(start)

	// 并行：约 100ms（a 几乎瞬时 + b/c 并行）；串行需要约 200ms
	if elapsed > 180*time.Millisecond {
		t.Fatalf("expected parallel start (~100ms), got %v", elapsed)
	}
}

func TestStart_CyclicDependencyDetected(t *testing.T) {
	a := newA("a", container.TypeOf[*compB]())
	b := newB("b", container.TypeOf[*compA]())

	lc := lifecycle.New()
	lc.Add(a).Add(b)

	err := lc.Start(context.Background())
	if err == nil {
		t.Fatalf("expected cyclic dependency error")
	}
	var cycErr *lifecycle.CyclicDependencyError
	if !errors.As(err, &cycErr) {
		t.Fatalf("expected *CyclicDependencyError, got %T: %v", err, err)
	}
	if len(cycErr.Cycle) < 2 {
		t.Fatalf("cycle should contain at least 2 names, got %v", cycErr.Cycle)
	}
}

func TestStart_FailureRollback(t *testing.T) {
	a := newA("a")
	b := newB("b", container.TypeOf[*compA]())
	b.startErr = errors.New("boom")

	lc := lifecycle.New()
	lc.Add(a).Add(b)

	err := lc.Start(context.Background())
	if err == nil {
		t.Fatalf("expected error from b.Start")
	}

	// a 启动成功，b 启动失败 → a 应被 Stop（回滚）
	if !a.stopped.Load() {
		t.Fatalf("expected a to be stopped during rollback")
	}
	// b 启动失败本身没成功启动，可能未被加入 started，因此可能不会被 Stop
	// 不强约束 b.stopCount
}

func TestStop_AggregatesErrors(t *testing.T) {
	a := newA("a")
	b := newB("b", container.TypeOf[*compA]())
	a.stopErr = errors.New("stop-a-err")
	b.stopErr = errors.New("stop-b-err")

	lc := lifecycle.New()
	lc.Add(a).Add(b)
	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	err := lc.Stop(context.Background())
	if err == nil {
		t.Fatalf("expected aggregated stop errors")
	}
	msg := err.Error()
	if !contains(msg, "stop-a-err") || !contains(msg, "stop-b-err") {
		t.Fatalf("both errors should be in aggregated message, got: %s", msg)
	}
}

func TestStop_PerComponentTimeout(t *testing.T) {
	a := newA("a")
	a.stopDelay = 500 * time.Millisecond

	lc := lifecycle.New(lifecycle.WithStopTimeoutPerComponent(50 * time.Millisecond))
	lc.Add(a)
	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	start := time.Now()
	err := lc.Stop(context.Background())
	elapsed := time.Since(start)
	if elapsed > 150*time.Millisecond {
		t.Fatalf("Stop should respect per-component timeout, took %v", elapsed)
	}
	// Stop 函数本身可能记录 deadline exceeded 错误
	_ = err
}

func TestHealth_OK(t *testing.T) {
	a := newA("a")
	b := newB("b")

	lc := lifecycle.New()
	lc.Add(a).Add(b)
	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	rep := lc.Health(context.Background())
	if rep.Status != lifecycle.HealthOK {
		t.Fatalf("expected HealthOK, got %s", rep.Status)
	}
	if len(rep.Details) != 2 {
		t.Fatalf("expected 2 components in report, got %d", len(rep.Details))
	}
}

func TestHealth_Degraded(t *testing.T) {
	a := newA("a")
	b := newB("b")
	b.healthErr = errors.New("unhealthy")

	lc := lifecycle.New()
	lc.Add(a).Add(b)
	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	rep := lc.Health(context.Background())
	if rep.Status != lifecycle.HealthDegraded {
		t.Fatalf("expected HealthDegraded, got %s", rep.Status)
	}
	if rep.Details["b"] == nil {
		t.Fatalf("expected b to have non-nil error in report")
	}
	if rep.Details["a"] != nil {
		t.Fatalf("expected a to be nil in report")
	}
}

func TestCollectDeps(t *testing.T) {
	a := newA("a")
	b := newB("b", container.TypeOf[*compA]())

	lc := lifecycle.New()
	lc.Add(a).Add(b)

	deps := lc.CollectDeps()
	if len(deps["b"]) != 1 {
		t.Fatalf("expected b to have 1 dep, got %v", deps)
	}
}

func TestMissingDependency(t *testing.T) {
	// a 依赖 b，但 b 未注册
	a := newA("a", container.TypeOf[*compB]())
	lc := lifecycle.New()
	lc.Add(a)
	err := lc.Start(context.Background())
	if err == nil {
		t.Fatalf("expected missing dep error")
	}
	var missErr *lifecycle.MissingDependencyError
	if !errors.As(err, &missErr) {
		t.Fatalf("expected *MissingDependencyError, got %T: %v", err, err)
	}
}

func TestConcurrentHealth_NoRace(t *testing.T) {
	a := newA("a")
	lc := lifecycle.New()
	lc.Add(a)
	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_ = lc.Health(context.Background())
		}()
	}
	wg.Wait()
}

// watcherComp 同时实现 Starter 和 Watcher
type watcherComp struct {
	fakeComp
	watchCalled atomic.Bool
}

func (w *watcherComp) Watch(_ context.Context) error {
	w.watchCalled.Store(true)
	return nil
}

func TestWatch_CalledAfterStart(t *testing.T) {
	w := &watcherComp{fakeComp: fakeComp{name: "w"}}
	lc := lifecycle.New()
	lc.Add(w)
	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	// 等待 Watch goroutine 完成（小睡眠）
	time.Sleep(20 * time.Millisecond)
	if !w.watchCalled.Load() {
		t.Fatalf("Watch should have been called")
	}
}

// initComp 实现 Initializer
type initComp struct {
	fakeComp
	initCalled atomic.Bool
	initErr    error
}

func (i *initComp) Init(_ context.Context) error {
	i.initCalled.Store(true)
	return i.initErr
}

func TestInit_CalledBeforeStart(t *testing.T) {
	i := &initComp{fakeComp: fakeComp{name: "i"}}
	lc := lifecycle.New()
	lc.Add(i)
	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	if !i.initCalled.Load() {
		t.Fatalf("Init should have been called")
	}
	if !i.started.Load() {
		t.Fatalf("Start should have been called")
	}
}

func TestInit_ErrorStopsStart(t *testing.T) {
	i := &initComp{fakeComp: fakeComp{name: "i"}}
	i.initErr = fmt.Errorf("init boom")
	lc := lifecycle.New()
	lc.Add(i)
	err := lc.Start(context.Background())
	if err == nil {
		t.Fatalf("expected Start to fail when Init fails")
	}
	if i.started.Load() {
		t.Fatalf("Start should not be called when Init fails")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

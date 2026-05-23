package next_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nextmicro/next"
	"github.com/nextmicro/next/container"
	"github.com/nextmicro/next/lifecycle"
	"github.com/nextmicro/next/middleware"
)

// fakeService 用于测试 WithContainer 注册。
type fakeService struct{ name string }

func (f *fakeService) Hello() string { return "hello-" + f.name }

func TestNew_DefaultsAndContainerRegistration(t *testing.T) {
	svc := &fakeService{name: "svc"}

	app, err := next.New(
		next.ID("test-id"),
		next.Name("test-name"),
		next.Version("v0.0.1"),
		// 显式空 components，避免触发默认 4 个基础设施 Component（依赖配置文件）
		next.WithComponents(),
		next.WithContainer(func(c *container.Container) {
			if err := container.Register(c, svc); err != nil {
				t.Fatalf("register fakeService: %v", err)
			}
		}),
	)
	if err != nil {
		t.Fatalf("next.New: %v", err)
	}

	if app.ID() != "test-id" {
		t.Errorf("ID = %q, want %q", app.ID(), "test-id")
	}
	if app.Name() != "test-name" {
		t.Errorf("Name = %q, want %q", app.Name(), "test-name")
	}
	if app.Version() != "v0.0.1" {
		t.Errorf("Version = %q, want %q", app.Version(), "v0.0.1")
	}

	got, err := container.Resolve[*fakeService](app.Container())
	if err != nil {
		t.Fatalf("Resolve fakeService: %v", err)
	}
	if got != svc {
		t.Errorf("Resolve returned different instance: got %p want %p", got, svc)
	}
	if got.Hello() != "hello-svc" {
		t.Errorf("Hello() = %q", got.Hello())
	}
}

func TestNew_MiddlewareRegistryWired(t *testing.T) {
	custom := middleware.NewRegistry()
	app, err := next.New(
		next.Name("mw-test"),
		next.WithComponents(),
		next.WithMiddlewareRegistry(custom),
	)
	if err != nil {
		t.Fatalf("next.New: %v", err)
	}

	if app.MiddlewareRegistry() != custom {
		t.Errorf("MiddlewareRegistry not injected into app")
	}
	resolved, err := container.Resolve[middleware.Registry](app.Container())
	if err != nil {
		t.Fatalf("Resolve middleware.Registry: %v", err)
	}
	if resolved != custom {
		t.Errorf("Container did not return injected Registry")
	}
}

// fakeComponent 验证 Lifecycle 编排被触发。
type fakeComponent struct {
	name      string
	startN    int32
	stopN     int32
	startSeen chan struct{}
}

func (f *fakeComponent) Name() string { return f.name }
func (f *fakeComponent) Start(_ context.Context) error {
	atomic.AddInt32(&f.startN, 1)
	close(f.startSeen)
	return nil
}
func (f *fakeComponent) Stop(_ context.Context) error {
	atomic.AddInt32(&f.stopN, 1)
	return nil
}

func TestApp_RunStartsLifecycleThenStopsOnContextCancel(t *testing.T) {
	comp := &fakeComponent{name: "fake", startSeen: make(chan struct{})}

	ctx, cancel := context.WithCancel(context.Background())
	app, err := next.New(
		next.Name("lifecycle-test"),
		next.Context(ctx),
		next.WithComponents(comp),
	)
	if err != nil {
		t.Fatalf("next.New: %v", err)
	}

	runErrCh := make(chan error, 1)
	go func() { runErrCh <- app.Run() }()

	select {
	case <-comp.startSeen:
	case <-time.After(2 * time.Second):
		t.Fatal("component Start was not invoked within 2s")
	}

	// 取消 ctx 触发 kratos.App 退出
	cancel()

	select {
	case err := <-runErrCh:
		// kratos.App.Run 在 ctx 取消时返回 context.Canceled（或 nil，取决于 kratos 版本）
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Logf("Run returned: %v (non-fatal)", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("app.Run did not return within 3s of ctx cancel")
	}

	if atomic.LoadInt32(&comp.startN) != 1 {
		t.Errorf("Start should be called exactly once, got %d", comp.startN)
	}
	if atomic.LoadInt32(&comp.stopN) != 1 {
		t.Errorf("Stop should be called exactly once, got %d", comp.stopN)
	}
}

func TestApp_ContainerAndLifecycleAccessors(t *testing.T) {
	app, err := next.New(
		next.Name("accessors"),
		next.WithComponents(),
	)
	if err != nil {
		t.Fatalf("next.New: %v", err)
	}
	if app.Container() == nil {
		t.Error("Container() returned nil")
	}
	if app.Lifecycle() == nil {
		t.Error("Lifecycle() returned nil")
	}
	if _, ok := any(app.Lifecycle()).(*lifecycle.Lifecycle); !ok {
		t.Error("Lifecycle() did not return *lifecycle.Lifecycle")
	}
}

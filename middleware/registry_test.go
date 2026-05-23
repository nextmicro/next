package middleware_test

import (
	"context"
	"errors"
	"testing"

	kmw "github.com/go-kratos/kratos/v2/middleware"
	configv1 "github.com/nextmicro/next/api/config/v1"
	"github.com/nextmicro/next/middleware"
)

// passthrough 返回一个透传的中间件，并在工厂调用时记录 name。
func passthrough(t *testing.T, observed *[]string) middleware.Factory {
	t.Helper()
	return func(c *configv1.Middleware) (kmw.Middleware, error) {
		*observed = append(*observed, c.GetName())
		return func(handler kmw.Handler) kmw.Handler {
			return func(ctx context.Context, req any) (any, error) {
				return handler(ctx, req)
			}
		}, nil
	}
}

func TestRegistry_Isolation(t *testing.T) {
	r1 := middleware.NewRegistry()
	r2 := middleware.NewRegistry()

	var observed1, observed2 []string
	r1.Register("server.tracing", passthrough(t, &observed1))
	r2.Register("server.logging", passthrough(t, &observed2))

	if _, err := r1.Create(&configv1.Middleware{Name: "server.tracing"}); err != nil {
		t.Fatalf("r1 should resolve server.tracing, got %v", err)
	}
	if _, err := r2.Create(&configv1.Middleware{Name: "server.tracing"}); !errors.Is(err, middleware.ErrNotFound) {
		t.Fatalf("r2 should not resolve r1's factory, got %v", err)
	}
}

func TestBuild_InputNotMutated(t *testing.T) {
	r := middleware.NewRegistry()
	var observed []string
	r.Register("server.tracing", passthrough(t, &observed))
	r.Register("server.logging", passthrough(t, &observed))

	configs := []*configv1.Middleware{
		{Name: "tracing"},
		{Name: "logging"},
	}
	// 保留指针引用，验证 Build 后元素 Name 未被修改
	original := []string{configs[0].Name, configs[1].Name}

	ms, err := r.Build("server", configs)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if len(ms) != 2 {
		t.Fatalf("expected 2 middleware, got %d", len(ms))
	}
	if configs[0].Name != original[0] || configs[1].Name != original[1] {
		t.Fatalf("Build mutated input: got %q,%q, want %q,%q",
			configs[0].Name, configs[1].Name, original[0], original[1])
	}
	if observed[0] != "tracing" || observed[1] != "logging" {
		t.Fatalf("factory observed wrong name: %v", observed)
	}
}

func TestBuild_PreservesOrder(t *testing.T) {
	r := middleware.NewRegistry()
	var observed []string
	r.Register("server.a", passthrough(t, &observed))
	r.Register("server.b", passthrough(t, &observed))
	r.Register("server.c", passthrough(t, &observed))

	configs := []*configv1.Middleware{
		{Name: "c"}, {Name: "a"}, {Name: "b"},
	}
	if _, err := r.Build("server", configs); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if observed[0] != "c" || observed[1] != "a" || observed[2] != "b" {
		t.Fatalf("Build did not preserve order: %v", observed)
	}
}

func TestBuild_UnknownReturnsError(t *testing.T) {
	r := middleware.NewRegistry()
	_, err := r.Build("server", []*configv1.Middleware{{Name: "unknown"}})
	if !errors.Is(err, middleware.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestBuild_WarnOnMissingSkips(t *testing.T) {
	r := middleware.NewRegistry()
	var observed []string
	r.Register("server.known", passthrough(t, &observed))

	var warned []string
	ms, err := r.BuildWithOptions("server", []*configv1.Middleware{
		{Name: "missing"},
		{Name: "known"},
	}, middleware.WarnOnMissing(func(name string) { warned = append(warned, name) }))

	if err != nil {
		t.Fatalf("WarnOnMissing should not return error, got %v", err)
	}
	if len(ms) != 1 {
		t.Fatalf("expected 1 middleware, got %d", len(ms))
	}
	if len(warned) != 1 || warned[0] != "server.missing" {
		t.Fatalf("expected warn for server.missing, got %v", warned)
	}
}

func TestBuild_ScopeFiltering(t *testing.T) {
	r := middleware.NewRegistry()
	var serverObserved, clientObserved []string
	r.Register("server.tracing", passthrough(t, &serverObserved))
	r.Register("client.tracing", passthrough(t, &clientObserved))

	cfgs := []*configv1.Middleware{{Name: "tracing"}}

	if _, err := r.Build("server", cfgs); err != nil {
		t.Fatalf("server build failed: %v", err)
	}
	if _, err := r.Build("client", cfgs); err != nil {
		t.Fatalf("client build failed: %v", err)
	}

	if len(serverObserved) != 1 || serverObserved[0] != "tracing" {
		t.Fatalf("server factory not invoked correctly: %v", serverObserved)
	}
	if len(clientObserved) != 1 || clientObserved[0] != "tracing" {
		t.Fatalf("client factory not invoked correctly: %v", clientObserved)
	}
}

func TestBuild_NameLookupCaseInsensitive(t *testing.T) {
	r := middleware.NewRegistry()
	var observed []string
	r.Register("Server.Tracing", passthrough(t, &observed))
	if _, err := r.Build("server", []*configv1.Middleware{{Name: "TRACING"}}); err != nil {
		t.Fatalf("expected case-insensitive lookup to succeed, got %v", err)
	}
}

func TestRegister_Overrides(t *testing.T) {
	r := middleware.NewRegistry()
	var a, b []string
	r.Register("server.x", passthrough(t, &a))
	r.Register("server.x", passthrough(t, &b)) // 覆盖
	_, _ = r.Create(&configv1.Middleware{Name: "server.x"})
	if len(a) != 0 || len(b) != 1 {
		t.Fatalf("Register did not override: a=%v b=%v", a, b)
	}
}

func TestFactory_ErrorPropagates(t *testing.T) {
	r := middleware.NewRegistry()
	r.Register("server.fail", func(c *configv1.Middleware) (kmw.Middleware, error) {
		return nil, errors.New("factory boom")
	})
	_, err := r.Build("server", []*configv1.Middleware{{Name: "fail"}})
	if err == nil {
		t.Fatalf("expected factory error to propagate")
	}
}

func TestCompat_BuildMiddleware_StripTransportPrefix(t *testing.T) {
	// 兼容旧 BuildMiddleware：传 "http.server" 应当映射到 scope="server"
	// 通过 DefaultRegistry 验证。
	var observed []string
	middleware.DefaultRegistry.Register("server.compat", passthrough(t, &observed))

	_, err := middleware.BuildMiddleware("http.server", []*configv1.Middleware{{Name: "compat"}})
	if err != nil {
		t.Fatalf("compat BuildMiddleware failed: %v", err)
	}
	if len(observed) != 1 {
		t.Fatalf("expected compat factory invoked once, got %v", observed)
	}
}

func TestCompat_BuildMiddleware_DoesNotMutateInput(t *testing.T) {
	middleware.DefaultRegistry.Register("server.original", passthrough(t, new([]string)))
	configs := []*configv1.Middleware{{Name: "original"}}
	originalName := configs[0].Name
	_, _ = middleware.BuildMiddleware("http.server", configs)
	if configs[0].Name != originalName {
		t.Fatalf("compat BuildMiddleware mutated input: %q -> %q", originalName, configs[0].Name)
	}
}

package transport_test

import (
	"context"
	"errors"
	"testing"

	kmw "github.com/go-kratos/kratos/v2/middleware"
	"google.golang.org/protobuf/types/known/durationpb"

	configv1 "github.com/nextmicro/next/api/config/v1"
	"github.com/nextmicro/next/middleware"
	"github.com/nextmicro/next/transport"
)

// noopMW 返回一个透传 middleware 并记录调用 name 到 observed。
func noopMW(observed *[]string) middleware.Factory {
	return func(c *configv1.Middleware) (kmw.Middleware, error) {
		*observed = append(*observed, c.GetName())
		return func(handler kmw.Handler) kmw.Handler {
			return func(ctx context.Context, req any) (any, error) {
				return handler(ctx, req)
			}
		}, nil
	}
}

func TestNewHTTPServer_NoConfigUsesDefaults(t *testing.T) {
	srv, err := transport.NewHTTPServer(transport.HTTPServerConfig{})
	if err != nil {
		t.Fatalf("NewHTTPServer: %v", err)
	}
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestNewHTTPServer_ConfigAppliesAddressTimeoutAndMiddleware(t *testing.T) {
	reg := middleware.NewRegistry()
	var observed []string
	reg.Register("server.tracing", noopMW(&observed))

	srv, err := transport.NewHTTPServer(transport.HTTPServerConfig{
		Registry: reg,
		ServerConfig: &configv1.HTTPServer{
			Network: "tcp",
			Addr:    "127.0.0.1:0",
			Timeout: durationpb.New(0),
			Middlewares: []*configv1.Middleware{
				{Name: "tracing"},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewHTTPServer: %v", err)
	}
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if len(observed) != 1 || observed[0] != "tracing" {
		t.Errorf("middleware factory not invoked correctly: %v", observed)
	}
}

func TestNewHTTPServer_MiddlewareErrorPropagates(t *testing.T) {
	reg := middleware.NewRegistry()
	_, err := transport.NewHTTPServer(transport.HTTPServerConfig{
		Registry: reg,
		ServerConfig: &configv1.HTTPServer{
			Middlewares: []*configv1.Middleware{{Name: "unknown"}},
		},
	})
	if !errors.Is(err, middleware.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestNewHTTPServer_RegistryNilFallsBackToDefault(t *testing.T) {
	middleware.DefaultRegistry.Register("server.fallback", noopMW(new([]string)))
	srv, err := transport.NewHTTPServer(transport.HTTPServerConfig{
		ServerConfig: &configv1.HTTPServer{
			Middlewares: []*configv1.Middleware{{Name: "fallback"}},
		},
	})
	if err != nil {
		t.Fatalf("NewHTTPServer: %v", err)
	}
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestNewGRPCServer_NoConfigUsesDefaults(t *testing.T) {
	srv, err := transport.NewGRPCServer(transport.GRPCServerConfig{})
	if err != nil {
		t.Fatalf("NewGRPCServer: %v", err)
	}
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestNewGRPCServer_ConfigAppliesMiddleware(t *testing.T) {
	reg := middleware.NewRegistry()
	var observed []string
	reg.Register("server.recovery", noopMW(&observed))

	srv, err := transport.NewGRPCServer(transport.GRPCServerConfig{
		Registry: reg,
		ServerConfig: &configv1.GRPCServer{
			Addr:    "127.0.0.1:0",
			Timeout: durationpb.New(0),
			Middlewares: []*configv1.Middleware{
				{Name: "recovery"},
			},
		},
	})
	if err != nil {
		t.Fatalf("NewGRPCServer: %v", err)
	}
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if len(observed) != 1 || observed[0] != "recovery" {
		t.Errorf("middleware factory not invoked correctly: %v", observed)
	}
}

func TestNewGRPCServer_MiddlewareErrorPropagates(t *testing.T) {
	reg := middleware.NewRegistry()
	_, err := transport.NewGRPCServer(transport.GRPCServerConfig{
		Registry: reg,
		ServerConfig: &configv1.GRPCServer{
			Middlewares: []*configv1.Middleware{{Name: "unknown"}},
		},
	})
	if !errors.Is(err, middleware.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMustNewHTTPServer_PanicsOnError(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	reg := middleware.NewRegistry()
	transport.MustNewHTTPServer(transport.HTTPServerConfig{
		Registry: reg,
		ServerConfig: &configv1.HTTPServer{
			Middlewares: []*configv1.Middleware{{Name: "unknown"}},
		},
	})
}

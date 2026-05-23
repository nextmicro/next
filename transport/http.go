// Package transport 提供基于 kratos 原版 transport 的薄装饰工厂。
//
// HTTP 路由 DSL 见 transport/httpx。
package transport

import (
	"fmt"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	configv1 "github.com/nextmicro/next/api/config/v1"
	"github.com/nextmicro/next/middleware"
)

// HTTPServerConfig 是 transport.NewHTTPServer 的可选行为开关。
type HTTPServerConfig struct {
	// Registry 用于装配配置中声明的 middleware（按 scope="server"）。
	// nil 时使用 middleware.DefaultRegistry。
	Registry middleware.Registry

	// ServerConfig 是 HTTP server 的配置；nil 表示沿用 kratos 默认。
	// 当 ServerConfig 与 ServerOptions 同时设置时，ServerOptions 覆盖配置中的同名字段。
	ServerConfig *configv1.HTTPServer

	// ServerOptions 直接透传给 khttp.NewServer。
	ServerOptions []khttp.ServerOption
}

// NewHTTPServer 基于配置 + 选项构造 kratos HTTP server，并自动装配 middleware 链。
//
// 装配顺序（middleware 链由外到内）：
//  1. 配置中声明的 middleware（按 scope="server"，由 Registry.Build 构建）
//  2. ServerOptions 中追加的 khttp.Middleware
//
// 返回的 *khttp.Server 可直接 RegisterService / HandleFunc，
// 然后通过 next.Server(srv) Option 挂载到 next.App。
//
// 注意：本工厂仅依赖 kratos 公开 API + next middleware.Registry。
func NewHTTPServer(cfg HTTPServerConfig) (*khttp.Server, error) {
	reg := cfg.Registry
	if reg == nil {
		reg = middleware.DefaultRegistry
	}

	opts := make([]khttp.ServerOption, 0, len(cfg.ServerOptions)+4)
	if c := cfg.ServerConfig; c != nil {
		if v := c.GetNetwork(); v != "" {
			opts = append(opts, khttp.Network(v))
		}
		if v := c.GetAddr(); v != "" {
			opts = append(opts, khttp.Address(v))
		}
		if v := c.GetTimeout().AsDuration(); v > 0 {
			opts = append(opts, khttp.Timeout(v))
		}
		if mws := c.GetMiddlewares(); len(mws) > 0 {
			built, err := reg.Build("server", mws)
			if err != nil {
				return nil, fmt.Errorf("transport.NewHTTPServer: build middlewares: %w", err)
			}
			if len(built) > 0 {
				opts = append(opts, khttp.Middleware(built...))
			}
		}
	}

	opts = append(opts, cfg.ServerOptions...)
	srv := khttp.NewServer(opts...)
	return srv, nil
}

// MustNewHTTPServer 是 NewHTTPServer 的 panic 版本，便于初始化失败直接 fail-fast。
func MustNewHTTPServer(cfg HTTPServerConfig) *khttp.Server {
	srv, err := NewHTTPServer(cfg)
	if err != nil {
		panic(err)
	}
	return srv
}

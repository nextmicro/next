package transport

import (
	"fmt"

	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"
	configv1 "github.com/nextmicro/next/api/config/v1"
	"github.com/nextmicro/next/middleware"
)

// GRPCServerConfig 是 transport.NewGRPCServer 的可选行为开关。
type GRPCServerConfig struct {
	// Registry 用于装配配置中声明的 middleware（按 scope="server"）。
	// nil 时使用 middleware.DefaultRegistry。
	Registry middleware.Registry

	// ServerConfig 是 gRPC server 的配置；nil 表示沿用 kratos 默认。
	// 当 ServerConfig 与 ServerOptions 同时设置时，ServerOptions 覆盖配置中的同名字段。
	ServerConfig *configv1.GRPCServer

	// ServerOptions 直接透传给 kgrpc.NewServer。
	ServerOptions []kgrpc.ServerOption
}

// NewGRPCServer 基于配置 + 选项构造 kratos gRPC server，并自动装配 middleware 链。
//
// 装配顺序：
//  1. 配置中声明的 middleware（按 scope="server"）
//  2. ServerOptions 中追加的 kgrpc.Middleware
//
// 返回的 *kgrpc.Server 可直接 RegisterService，然后通过 next.Server(srv) Option 挂载。
func NewGRPCServer(cfg GRPCServerConfig) (*kgrpc.Server, error) {
	reg := cfg.Registry
	if reg == nil {
		reg = middleware.DefaultRegistry
	}

	opts := make([]kgrpc.ServerOption, 0, len(cfg.ServerOptions)+4)
	if c := cfg.ServerConfig; c != nil {
		if v := c.GetNetwork(); v != "" {
			opts = append(opts, kgrpc.Network(v))
		}
		if v := c.GetAddr(); v != "" {
			opts = append(opts, kgrpc.Address(v))
		}
		if v := c.GetTimeout().AsDuration(); v > 0 {
			opts = append(opts, kgrpc.Timeout(v))
		}
		if mws := c.GetMiddlewares(); len(mws) > 0 {
			built, err := reg.Build("server", mws)
			if err != nil {
				return nil, fmt.Errorf("transport.NewGRPCServer: build middlewares: %w", err)
			}
			if len(built) > 0 {
				opts = append(opts, kgrpc.Middleware(built...))
			}
		}
	}

	opts = append(opts, cfg.ServerOptions...)
	srv := kgrpc.NewServer(opts...)
	return srv, nil
}

// MustNewGRPCServer 是 NewGRPCServer 的 panic 版本。
func MustNewGRPCServer(cfg GRPCServerConfig) *kgrpc.Server {
	srv, err := NewGRPCServer(cfg)
	if err != nil {
		panic(err)
	}
	return srv
}

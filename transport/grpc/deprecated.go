// Package grpc 提供 v1 兼容层，底层已迁移至 kratos 原生 gRPC transport。
//
// Deprecated: 新业务请使用 transport.NewGRPCServer。
package grpc

import (
	kgrpc "github.com/go-kratos/kratos/v2/transport/grpc"

	// 保留 next 自定义 resolver scheme 注册。
	_ "github.com/nextmicro/next/transport/grpc/resolver/direct"
)

// Server is the gRPC server type.
//
// Deprecated: use kgrpc.Server via transport.NewGRPCServer.
type Server = kgrpc.Server

// ServerOption configures the gRPC server.
//
// Deprecated: use kgrpc.ServerOption.
type ServerOption = kgrpc.ServerOption

// ClientOption configures the gRPC client.
//
// Deprecated: use kgrpc.ClientOption.
type ClientOption = kgrpc.ClientOption

// NewServer creates a gRPC server.
//
// Deprecated: use transport.NewGRPCServer or kgrpc.NewServer.
var NewServer = kgrpc.NewServer

// Dial creates a gRPC client connection.
//
// Deprecated: use kgrpc.Dial.
var Dial = kgrpc.Dial

// DialInsecure creates an insecure gRPC client connection.
//
// Deprecated: use kgrpc.DialInsecure.
var DialInsecure = kgrpc.DialInsecure

// Re-export common server options.

// Network sets server network.
var Network = kgrpc.Network

// Address sets server listen address.
var Address = kgrpc.Address

// Timeout sets server timeout.
var Timeout = kgrpc.Timeout

// Middleware adds service middleware.
var Middleware = kgrpc.Middleware

// TLSConfig sets TLS config.
var TLSConfig = kgrpc.TLSConfig

// Listener sets a custom listener.
var Listener = kgrpc.Listener

// Re-export common client options.

// WithEndpoint sets client endpoint.
var WithEndpoint = kgrpc.WithEndpoint

// WithTimeout sets client timeout.
var WithTimeout = kgrpc.WithTimeout

// WithMiddleware adds client middleware.
var WithMiddleware = kgrpc.WithMiddleware

// WithDiscovery sets service discovery.
var WithDiscovery = kgrpc.WithDiscovery

// WithTLSConfig sets client TLS config.
var WithTLSConfig = kgrpc.WithTLSConfig

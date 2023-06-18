package grpc

import (
	"github.com/go-kratos/kratos/v2/transport/grpc"
	conf "github.com/nextmicro/next/config"
	"github.com/nextmicro/next/transport"
)

type Server struct {
	*grpc.Server

	opt transport.Options
}

// NewServer creates an HTTP server by options.
func NewServer(opts ...transport.ServerOption) *Server {
	o, _ := transport.NewDefaultOptions(conf.AppConfig(), opts...)

	return &Server{
		Server: grpc.NewServer(
			grpc.Address(o.Address),
			grpc.Timeout(o.Timeout),
			grpc.Middleware(o.Middleware...),
		),
	}
}

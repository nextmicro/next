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
	o, _ := transport.NewDefaultOptions(conf.ApplicationConfig(), opts...)

	s := &Server{
		opt: *o,
	}

	s.Server = grpc.NewServer(s.buildOptions()...)
	return s
}

// buildOptions builds the http server options.
func (s *Server) buildOptions() []grpc.ServerOption {
	cfg := conf.ApplicationConfig().GetServer().GetHttp()
	var opts = make([]grpc.ServerOption, 0, 2)
	if s.opt.Address == "" && cfg.GetAddr() != "" {
		s.opt.Address = cfg.GetAddr()
		opts = append(opts, grpc.Address(s.opt.Address))
	}
	if s.opt.Timeout == 0 && cfg.GetTimeout().AsDuration() != 0 {
		s.opt.Timeout = cfg.GetTimeout().AsDuration()
		opts = append(opts, grpc.Timeout(s.opt.Timeout))
	}

	return opts
}

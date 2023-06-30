package http

import (
	chain "github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport/http"
	conf "github.com/nextmicro/next/config"
	"github.com/nextmicro/next/middleware"
	"github.com/nextmicro/next/transport"
)

type Server struct {
	*http.Server

	opt transport.Options
}

// NewServer creates an HTTP server by options.
func NewServer(opts ...transport.ServerOption) *Server {
	o, _ := transport.NewDefaultOptions(conf.ApplicationConfig(), opts...)
	s := &Server{
		opt: *o,
	}

	s.Server = http.NewServer(s.buildOptions()...)
	return s
}

// buildOptions builds the http server options.
func (s *Server) buildOptions() []http.ServerOption {
	cfg := conf.ApplicationConfig().GetServer().GetHttp()
	var opts = make([]http.ServerOption, 0, 2)
	if s.opt.Address == "" && cfg.GetAddr() != "" {
		s.opt.Address = cfg.GetAddr()
		opts = append(opts, http.Address(s.opt.Address))
	}
	if s.opt.Timeout == 0 && cfg.GetTimeout().AsDuration() != 0 {
		s.opt.Timeout = cfg.GetTimeout().AsDuration()
		opts = append(opts, http.Timeout(s.opt.Timeout))
	}
	// 将全局中间件放在最前面，然后是用户自定义的中间件
	ms := make([]chain.Middleware, 0, len(s.opt.Middleware)+len(cfg.GetMiddlewares()))
	if cfg != nil && cfg.GetMiddlewares() != nil {
		serverMs, _ := middleware.BuildMiddleware("server", cfg.GetMiddlewares())
		ms = append(ms, serverMs...)
	}
	if s.opt.Middleware != nil {
		ms = append(ms, s.opt.Middleware...)
	}

	if len(ms) > 0 {
		opts = append(opts, http.Middleware(ms...))
	}

	return opts
}

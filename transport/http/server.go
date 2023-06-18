package http

import (
	"github.com/go-kratos/kratos/v2/transport/http"
	conf "github.com/nextmicro/next/config"
	"github.com/nextmicro/next/transport"
)

type Server struct {
	*http.Server

	opt transport.Options
}

// NewServer creates an HTTP server by options.
func NewServer(opts ...transport.ServerOption) *Server {
	o, _ := transport.NewDefaultOptions(conf.AppConfig(), opts...)

	return &Server{
		Server: http.NewServer(
			http.Address(o.Address),
			http.Timeout(o.Timeout),
			http.Middleware(o.Middleware...),
		),
	}
}

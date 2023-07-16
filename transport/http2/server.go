package http2

import (
	"crypto/tls"
	chain "github.com/go-kratos/kratos/v2/middleware"
	tr "github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/gorilla/mux"
	conf "github.com/nextmicro/next/config"
	"github.com/nextmicro/next/internal/matcher"
	"github.com/nextmicro/next/middleware"
	"github.com/nextmicro/next/transport"
	"net"
	http2 "net/http"
	"net/url"
	"time"
)

var (
	_ tr.Server     = (*Server)(nil)
	_ tr.Endpointer = (*Server)(nil)
	_ http2.Handler = (*Server)(nil)
)

// ServerOption is an HTTP server option.
type ServerOption func(*Server)

// Network with server network.
func Network(network string) ServerOption {
	return func(s *Server) {
		s.network = network
	}
}

// Address with server address.
func Address(addr string) ServerOption {
	return func(s *Server) {
		s.address = addr
	}
}

// Timeout with server timeout.
func Timeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.timeout = timeout
	}
}

// Middleware with service middleware option.
func Middleware(m ...chain.Middleware) ServerOption {
	return func(o *Server) {
		o.middleware.Use(m...)
	}
}

// Filter with HTTP middleware option.
func Filter(filters ...http.FilterFunc) ServerOption {
	return func(o *Server) {
		o.filters = filters
	}
}

// RequestVarsDecoder with request decoder.
func RequestVarsDecoder(dec http.DecodeRequestFunc) ServerOption {
	return func(o *Server) {
		o.decVars = dec
	}
}

// RequestQueryDecoder with request decoder.
func RequestQueryDecoder(dec http.DecodeRequestFunc) ServerOption {
	return func(o *Server) {
		o.decQuery = dec
	}
}

// RequestDecoder with request decoder.
func RequestDecoder(dec http.DecodeRequestFunc) ServerOption {
	return func(o *Server) {
		o.decBody = dec
	}
}

// ResponseEncoder with response encoder.
func ResponseEncoder(en http.EncodeResponseFunc) ServerOption {
	return func(o *Server) {
		o.enc = en
	}
}

// ErrorEncoder with error encoder.
func ErrorEncoder(en http.EncodeErrorFunc) ServerOption {
	return func(o *Server) {
		o.ene = en
	}
}

// TLSConfig with TLS config.
func TLSConfig(c *tls.Config) ServerOption {
	return func(o *Server) {
		o.tlsConf = c
	}
}

// StrictSlash is with mux's StrictSlash
// If true, when the path pattern is "/path/", accessing "/path" will
// redirect to the former and vice versa.
func StrictSlash(strictSlash bool) ServerOption {
	return func(o *Server) {
		o.strictSlash = strictSlash
	}
}

// Listener with server lis
func Listener(lis net.Listener) ServerOption {
	return func(s *Server) {
		s.lis = lis
	}
}

// PathPrefix with mux's PathPrefix, router will replaced by a subrouter that start with prefix.
func PathPrefix(prefix string) ServerOption {
	return func(s *Server) {
		s.router = s.router.PathPrefix(prefix).Subrouter()
	}
}

type Server struct {
	*http2.Server
	lis         net.Listener
	tlsConf     *tls.Config
	endpoint    *url.URL
	err         error
	network     string
	address     string
	timeout     time.Duration
	filters     []http.FilterFunc
	middleware  matcher.Matcher
	decVars     http.DecodeRequestFunc
	decQuery    http.DecodeRequestFunc
	decBody     http.DecodeRequestFunc
	enc         http.EncodeResponseFunc
	ene         http.EncodeErrorFunc
	strictSlash bool
	router      *mux.Router

	opt transport.Options
}

// NewServer creates an HTTP server by options.
func NewServer(opts ...transport.ServerOption) *Server {
	o, _ := transport.NewDefaultOptions(conf.ApplicationConfig(), opts...)
	srv := &Server{
		opt:         *o,
		network:     "tcp",
		address:     ":0",
		timeout:     1 * time.Second,
		middleware:  matcher.New(),
		decVars:     http.DefaultRequestVars,
		decQuery:    http.DefaultRequestQuery,
		decBody:     http.DefaultRequestDecoder,
		enc:         http.DefaultResponseEncoder,
		ene:         http.DefaultErrorEncoder,
		strictSlash: true,
		router:      mux.NewRouter(),
	}

	srv.router.StrictSlash(srv.strictSlash)
	srv.router.NotFoundHandler = http2.DefaultServeMux
	srv.router.MethodNotAllowedHandler = http2.DefaultServeMux
	srv.router.Use(srv.filter())
	srv.Server = &http2.Server{
		Handler:   http.FilterChain(srv.filters...)(srv.router),
		TLSConfig: srv.tlsConf,
	}

	return srv
}

// Route registers an HTTP router.
func (s *Server) Route(prefix string, filters ...http.FilterFunc) *Router {

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

package grpc

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"time"

	chain "github.com/go-kratos/kratos/v2/middleware"
	conf "github.com/nextmicro/next/config"
	"github.com/nextmicro/next/internal/endpoint"
	"github.com/nextmicro/next/internal/host"
	"github.com/nextmicro/next/internal/matcher"
	middleware2 "github.com/nextmicro/next/middleware"

	"google.golang.org/grpc"
	"google.golang.org/grpc/admin"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	apimd "github.com/go-kratos/kratos/v2/api/metadata"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

var (
	_ transport.Server     = (*Server)(nil)
	_ transport.Endpointer = (*Server)(nil)
)

// ServerOption is gRPC server option.
type ServerOption func(o *Server)

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

// Endpoint with server address.
func Endpoint(endpoint *url.URL) ServerOption {
	return func(s *Server) {
		s.endpoint = endpoint
	}
}

// Timeout with server timeout.
func Timeout(timeout time.Duration) ServerOption {
	return func(s *Server) {
		s.timeout = timeout
	}
}

// Logger with server logger.
// Deprecated: use global logger instead.
func Logger(_ log.Logger) ServerOption {
	return func(s *Server) {}
}

// Middleware with server middleware.
func Middleware(m ...middleware.Middleware) ServerOption {
	return func(s *Server) {
		s.matcher.Use(m...)
	}
}

// CustomHealth Checks server.
func CustomHealth() ServerOption {
	return func(s *Server) {
		s.customHealth = true
	}
}

// TLSConfig with TLS config.
func TLSConfig(c *tls.Config) ServerOption {
	return func(s *Server) {
		s.tlsConf = c
	}
}

// Listener with server lis
func Listener(lis net.Listener) ServerOption {
	return func(s *Server) {
		s.lis = lis
	}
}

// UnaryInterceptor returns a ServerOption that sets the UnaryServerInterceptor for the server.
func UnaryInterceptor(in ...grpc.UnaryServerInterceptor) ServerOption {
	return func(s *Server) {
		s.unaryInts = in
	}
}

// StreamInterceptor returns a ServerOption that sets the StreamServerInterceptor for the server.
func StreamInterceptor(in ...grpc.StreamServerInterceptor) ServerOption {
	return func(s *Server) {
		s.streamInts = in
	}
}

// Options with grpc options.
func Options(opts ...grpc.ServerOption) ServerOption {
	return func(s *Server) {
		s.grpcOpts = opts
	}
}

// Server is a gRPC server wrapper.
type Server struct {
	*grpc.Server
	baseCtx      context.Context
	tlsConf      *tls.Config
	lis          net.Listener
	err          error
	network      string
	address      string
	endpoint     *url.URL
	timeout      time.Duration
	matcher      matcher.Matcher
	middleware   []middleware.Middleware
	unaryInts    []grpc.UnaryServerInterceptor
	streamInts   []grpc.StreamServerInterceptor
	grpcOpts     []grpc.ServerOption
	health       *health.Server
	customHealth bool
	metadata     *apimd.Server
	adminClean   func()
}

// NewServer creates a gRPC server by options.
func NewServer(opts ...ServerOption) *Server {
	srv := &Server{
		baseCtx: context.Background(),
		network: "tcp",
		address: ":0",
		timeout: 1 * time.Second,
		health:  health.NewServer(),
		matcher: matcher.New(),
	}
	for _, o := range opts {
		o(srv)
	}
	serverMs := srv.buildMiddlewareOptions()
	// server middleware first
	if len(serverMs) > 0 {
		userMs := srv.middleware
		srv.middleware = append(serverMs, userMs...)
	}
	srv.matcher.Use(srv.middleware...)

	unaryInts := []grpc.UnaryServerInterceptor{
		srv.unaryServerInterceptor(),
	}
	streamInts := []grpc.StreamServerInterceptor{
		srv.streamServerInterceptor(),
	}
	if len(srv.unaryInts) > 0 {
		unaryInts = append(unaryInts, srv.unaryInts...)
	}
	if len(srv.streamInts) > 0 {
		streamInts = append(streamInts, srv.streamInts...)
	}
	grpcOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(unaryInts...),
		grpc.ChainStreamInterceptor(streamInts...),
	}
	if srv.tlsConf != nil {
		grpcOpts = append(grpcOpts, grpc.Creds(credentials.NewTLS(srv.tlsConf)))
	}
	if len(srv.grpcOpts) > 0 {
		grpcOpts = append(grpcOpts, srv.grpcOpts...)
	}
	srv.Server = grpc.NewServer(grpcOpts...)
	srv.metadata = apimd.NewServer(srv.Server)
	// internal register
	if !srv.customHealth {
		grpc_health_v1.RegisterHealthServer(srv.Server, srv.health)
	}
	apimd.RegisterMetadataServer(srv.Server, srv.metadata)
	reflection.Register(srv.Server)
	// admin register
	srv.adminClean, _ = admin.Register(srv.Server)
	return srv
}

// buildMiddlewareOptions builds the http server options.
func (s *Server) buildMiddlewareOptions() []middleware.Middleware {
	cfg := conf.ApplicationConfig().GetServer().GetGrpc()
	if cfg.GetAddr() != "" {
		s.address = cfg.GetAddr()
	}
	if cfg.GetNetwork() != "" {
		s.network = cfg.GetNetwork()
	}
	if cfg.GetTimeout().AsDuration() != 0 {
		s.timeout = cfg.GetTimeout().AsDuration()
	}

	ms := make([]chain.Middleware, 0, len(cfg.GetMiddlewares()))
	if cfg != nil && cfg.GetMiddlewares() != nil {
		serverMs, _ := middleware2.BuildMiddleware("server", cfg.GetMiddlewares())
		ms = append(ms, serverMs...)
	}

	return ms
}

// Use uses a service middleware with selector.
// selector:
//   - '/*'
//   - '/helloworld.v1.Greeter/*'
//   - '/helloworld.v1.Greeter/SayHello'
func (s *Server) Use(selector string, m ...middleware.Middleware) {
	s.matcher.Add(selector, m...)
}

// Endpoint return a real address to registry endpoint.
// examples:
//
//	grpc://127.0.0.1:9000?isSecure=false
func (s *Server) Endpoint() (*url.URL, error) {
	if err := s.listenAndEndpoint(); err != nil {
		return nil, s.err
	}
	return s.endpoint, nil
}

// Start start the gRPC server.
func (s *Server) Start(ctx context.Context) error {
	if err := s.listenAndEndpoint(); err != nil {
		return s.err
	}
	s.baseCtx = ctx
	log.Infof("[gRPC] server listening on: %s", s.lis.Addr().String())
	s.health.Resume()
	return s.Serve(s.lis)
}

// Stop stop the gRPC server.
func (s *Server) Stop(_ context.Context) error {
	if s.adminClean != nil {
		s.adminClean()
	}
	s.health.Shutdown()
	s.GracefulStop()
	log.Info("[gRPC] server stopping")
	return nil
}

func (s *Server) listenAndEndpoint() error {
	if s.lis == nil {
		lis, err := net.Listen(s.network, s.address)
		if err != nil {
			s.err = err
			return err
		}
		s.lis = lis
	}
	if s.endpoint == nil {
		addr, err := host.Extract(s.address, s.lis)
		if err != nil {
			s.err = err
			return err
		}
		s.endpoint = endpoint.NewEndpoint(endpoint.Scheme("grpc", s.tlsConf != nil), addr)
	}
	return s.err
}

package grpc

import (
	"context"
	"crypto/tls"
	"time"

	chain "github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
	client "github.com/go-kratos/kratos/v2/transport/grpc"
	v1 "github.com/nextmicro/next/api/config/v1"
	"github.com/nextmicro/next/middleware"
	"google.golang.org/grpc"
)

// ClientOption is gRPC client option.
type ClientOption func(o *clientOptions)

// WithEndpoint with client endpoint.
func WithEndpoint(endpoint string) ClientOption {
	return func(o *clientOptions) {
		o.endpoint = endpoint
	}
}

// WithSubset with client disocvery subset size.
// zero value means subset filter disabled
func WithSubset(size int) ClientOption {
	return func(o *clientOptions) {
		o.subsetSize = size
	}
}

// WithTimeout with client timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.timeout = timeout
	}
}

// WithMiddleware with client middleware.
func WithMiddleware(m ...chain.Middleware) ClientOption {
	return func(o *clientOptions) {
		o.middleware = m
	}
}

// WithDiscovery with client discovery.
func WithDiscovery(d registry.Discovery) ClientOption {
	return func(o *clientOptions) {
		o.discovery = d
	}
}

// WithTLSConfig with TLS config.
func WithTLSConfig(c *tls.Config) ClientOption {
	return func(o *clientOptions) {
		o.tlsConf = c
	}
}

// WithUnaryInterceptor returns a DialOption that specifies the interceptor for unary RPCs.
func WithUnaryInterceptor(in ...grpc.UnaryClientInterceptor) ClientOption {
	return func(o *clientOptions) {
		o.ints = in
	}
}

// WithStreamInterceptor returns a DialOption that specifies the interceptor for streaming RPCs.
func WithStreamInterceptor(in ...grpc.StreamClientInterceptor) ClientOption {
	return func(o *clientOptions) {
		o.streamInts = in
	}
}

// WithOptions with gRPC options.
func WithOptions(opts ...grpc.DialOption) ClientOption {
	return func(o *clientOptions) {
		o.grpcOpts = opts
	}
}

// WithNodeFilter with select filters
func WithNodeFilter(filters ...selector.NodeFilter) ClientOption {
	return func(o *clientOptions) {
		o.filters = filters
	}
}

// WithPrintDiscoveryDebugLog with print discovery debug log
func WithPrintDiscoveryDebugLog(p bool) ClientOption {
	return func(o *clientOptions) {
		o.printDiscoveryDebugLog = p
	}
}

// WithContext with client context.
func WithContext(ctx context.Context) ClientOption {
	return func(o *clientOptions) {
		o.ctx = ctx
	}
}

// clientOptions is gRPC Client
type clientOptions struct {
	ctx                    context.Context
	endpoint               string
	subsetSize             int
	tlsConf                *tls.Config
	timeout                time.Duration
	discovery              registry.Discovery
	middleware             []chain.Middleware
	ints                   []grpc.UnaryClientInterceptor
	streamInts             []grpc.StreamClientInterceptor
	grpcOpts               []grpc.DialOption
	balancerName           string
	filters                []selector.NodeFilter
	printDiscoveryDebugLog bool
}

type Client struct {
	opts clientOptions
	*grpc.ClientConn
}

func NewClient(cfg *v1.GRPCClient, opts ...ClientOption) (*Client, error) {
	options := clientOptions{
		ctx:                    context.Background(),
		timeout:                2000 * time.Millisecond,
		balancerName:           "selector",
		subsetSize:             25,
		printDiscoveryDebugLog: true,
	}
	for _, o := range opts {
		o(&options)
	}

	c := &Client{
		opts: options,
	}

	conn, err := client.DialInsecure(options.ctx, c.buildDialOptions(cfg, options)...)
	if err != nil {
		return nil, err
	}
	c.ClientConn = conn

	return c, nil
}

// buildDialOptions build dial options.
func (c *Client) buildDialOptions(cfg *v1.GRPCClient, opts clientOptions) []client.ClientOption {
	options := make([]client.ClientOption, 0, 10)
	// 将全局中间件放在最前面，然后是用户自定义的中间件
	ms := make([]chain.Middleware, 0, len(opts.middleware)+len(cfg.GetMiddlewares()))
	if cfg != nil && cfg.GetMiddlewares() != nil {
		serverMs, _ := middleware.BuildMiddleware(cfg.GetMiddlewares())
		ms = append(ms, serverMs...)
	}
	if opts.middleware != nil {
		ms = append(ms, opts.middleware...)
	}
	if len(ms) > 0 {
		options = append(options, client.WithMiddleware(ms...))
	}
	if opts.endpoint != "" {
		options = append(options, client.WithEndpoint(opts.endpoint))
	}
	if opts.subsetSize > 0 {
		options = append(options, client.WithSubset(opts.subsetSize))
	}
	if opts.timeout > 0 {
		options = append(options, client.WithTimeout(opts.timeout))
	}
	if opts.discovery != nil {
		options = append(options, client.WithDiscovery(opts.discovery))
	}
	if opts.tlsConf != nil {
		options = append(options, client.WithTLSConfig(opts.tlsConf))
	}
	if opts.ints != nil {
		options = append(options, client.WithUnaryInterceptor(opts.ints...))
	}
	if opts.streamInts != nil {
		options = append(options, client.WithStreamInterceptor(opts.streamInts...))
	}
	if opts.grpcOpts != nil {
		options = append(options, client.WithOptions(opts.grpcOpts...))
	}
	if opts.filters != nil {
		options = append(options, client.WithNodeFilter(opts.filters...))
	}
	if opts.printDiscoveryDebugLog {
		options = append(options, client.WithPrintDiscoveryDebugLog(opts.printDiscoveryDebugLog))
	}

	return options
}

package http

import (
	"context"
	"crypto/tls"
	http2 "net/http"
	"time"

	chain "github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/transport/http"
	v1 "github.com/nextmicro/next/api/config/v1"
	"github.com/nextmicro/next/middleware"
)

// ClientOption is HTTP client option.
type ClientOption func(*clientOptions)

// Client is an HTTP transport client.
type clientOptions struct {
	ctx          context.Context
	tlsConf      *tls.Config
	timeout      time.Duration
	endpoint     string
	userAgent    string
	encoder      http.EncodeRequestFunc
	decoder      http.DecodeResponseFunc
	errorDecoder http.DecodeErrorFunc
	transport    http2.RoundTripper
	nodeFilters  []selector.NodeFilter
	discovery    registry.Discovery
	middleware   []chain.Middleware
	block        bool
	subsetSize   int
}

// WithSubset with client disocvery subset size.
// zero value means subset filter disabled
func WithSubset(size int) ClientOption {
	return func(o *clientOptions) {
		o.subsetSize = size
	}
}

// WithTransport with client transport.
func WithTransport(trans http2.RoundTripper) ClientOption {
	return func(o *clientOptions) {
		o.transport = trans
	}
}

// WithTimeout with client request timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.timeout = d
	}
}

// WithUserAgent with client user agent.
func WithUserAgent(ua string) ClientOption {
	return func(o *clientOptions) {
		o.userAgent = ua
	}
}

// WithMiddleware with client middleware.
func WithMiddleware(m ...chain.Middleware) ClientOption {
	return func(o *clientOptions) {
		o.middleware = m
	}
}

// WithEndpoint with client addr.
func WithEndpoint(endpoint string) ClientOption {
	return func(o *clientOptions) {
		o.endpoint = endpoint
	}
}

// WithRequestEncoder with client request encoder.
func WithRequestEncoder(encoder http.EncodeRequestFunc) ClientOption {
	return func(o *clientOptions) {
		o.encoder = encoder
	}
}

// WithResponseDecoder with client response decoder.
func WithResponseDecoder(decoder http.DecodeResponseFunc) ClientOption {
	return func(o *clientOptions) {
		o.decoder = decoder
	}
}

// WithErrorDecoder with client error decoder.
func WithErrorDecoder(errorDecoder http.DecodeErrorFunc) ClientOption {
	return func(o *clientOptions) {
		o.errorDecoder = errorDecoder
	}
}

// WithDiscovery with client discovery.
func WithDiscovery(d registry.Discovery) ClientOption {
	return func(o *clientOptions) {
		o.discovery = d
	}
}

// WithNodeFilter with select filters
func WithNodeFilter(filters ...selector.NodeFilter) ClientOption {
	return func(o *clientOptions) {
		o.nodeFilters = filters
	}
}

// WithBlock with client block.
func WithBlock() ClientOption {
	return func(o *clientOptions) {
		o.block = true
	}
}

// WithTLSConfig with tls config.
func WithTLSConfig(c *tls.Config) ClientOption {
	return func(o *clientOptions) {
		o.tlsConf = c
	}
}

// WithContext with client context.
func WithContext(ctx context.Context) ClientOption {
	return func(o *clientOptions) {
		o.ctx = ctx
	}
}

type Client struct {
	opts clientOptions
	*http.Client
}

// NewClient returns an HTTP client.
func NewClient(cfg *v1.HTTPClient, opts ...ClientOption) (*Client, error) {
	var (
		err     error
		options = clientOptions{
			ctx:          context.Background(),
			timeout:      2000 * time.Millisecond,
			encoder:      http.DefaultRequestEncoder,
			decoder:      http.DefaultResponseDecoder,
			errorDecoder: http.DefaultErrorDecoder,
			transport:    http2.DefaultTransport,
			subsetSize:   25,
		}
	)

	for _, o := range opts {
		o(&options)
	}

	c := &Client{
		opts: options,
	}

	c.Client, err = http.NewClient(options.ctx, c.buildDialOptions(cfg, options)...)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// buildDialOptions build dial options.
func (c *Client) buildDialOptions(cfg *v1.HTTPClient, opts clientOptions) []http.ClientOption {
	var options = make([]http.ClientOption, 0, 10)
	// 将全局中间件放在最前面，然后是用户自定义的中间件
	ms := make([]chain.Middleware, 0, len(opts.middleware)+len(cfg.GetMiddlewares()))
	if cfg != nil && cfg.GetMiddlewares() != nil {
		serverMs, _ := middleware.BuildMiddleware("client", cfg.GetMiddlewares())
		ms = append(ms, serverMs...)
	}
	if opts.middleware != nil {
		ms = append(ms, opts.middleware...)
	}
	if opts.tlsConf != nil {
		options = append(options, http.WithTLSConfig(opts.tlsConf))
	}
	if opts.timeout != 0 {
		options = append(options, http.WithTimeout(opts.timeout))
	}
	if opts.userAgent != "" {
		options = append(options, http.WithUserAgent(opts.userAgent))
	}
	if opts.encoder != nil {
		options = append(options, http.WithRequestEncoder(opts.encoder))
	}
	if opts.decoder != nil {
		options = append(options, http.WithResponseDecoder(opts.decoder))
	}
	if opts.errorDecoder != nil {
		options = append(options, http.WithErrorDecoder(opts.errorDecoder))
	}
	if opts.transport != nil {
		options = append(options, http.WithTransport(opts.transport))
	}
	if opts.endpoint != "" {
		options = append(options, http.WithEndpoint(opts.endpoint))
	}
	if opts.discovery != nil {
		options = append(options, http.WithDiscovery(opts.discovery))
	}
	if len(opts.nodeFilters) > 0 {
		options = append(options, http.WithNodeFilter(opts.nodeFilters...))
	}
	if opts.block {
		options = append(options, http.WithBlock())
	}
	if opts.subsetSize > 0 {
		options = append(options, http.WithSubset(opts.subsetSize))
	}
	if len(ms) > 0 {
		options = append(options, http.WithMiddleware(ms...))
	}
	return options
}

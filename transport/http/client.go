package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2"
	v1 "github.com/nextmicro/next/api/config/v1"
	"github.com/nextmicro/next/internal/host"
	"github.com/nextmicro/next/internal/httputil"
	chain "github.com/nextmicro/next/middleware"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/selector/wrr"
	"github.com/go-kratos/kratos/v2/transport"
)

func init() {
	if selector.GlobalSelector() == nil {
		selector.SetGlobalSelector(wrr.NewBuilder())
	}
}

// DecodeErrorFunc is decode error func.
type DecodeErrorFunc func(ctx context.Context, res *http.Response) error

// EncodeRequestFunc is request encode func.
type EncodeRequestFunc func(ctx context.Context, contentType string, in interface{}) (body []byte, err error)

// DecodeResponseFunc is response decode func.
type DecodeResponseFunc func(ctx context.Context, res *http.Response, out interface{}) error

// ClientOption is HTTP client option.
type ClientOption func(*clientOptions)

// Client is an HTTP transport client.
type clientOptions struct {
	cfg          *v1.HTTPClient
	ctx          context.Context
	tlsConf      *tls.Config
	timeout      time.Duration
	endpoint     string
	userAgent    string
	encoder      EncodeRequestFunc
	decoder      DecodeResponseFunc
	errorDecoder DecodeErrorFunc
	transport    http.RoundTripper
	nodeFilters  []selector.NodeFilter
	discovery    registry.Discovery
	middleware   []middleware.Middleware
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
func WithTransport(trans http.RoundTripper) ClientOption {
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
func WithMiddleware(m ...middleware.Middleware) ClientOption {
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
func WithRequestEncoder(encoder EncodeRequestFunc) ClientOption {
	return func(o *clientOptions) {
		o.encoder = encoder
	}
}

// WithResponseDecoder with client response decoder.
func WithResponseDecoder(decoder DecodeResponseFunc) ClientOption {
	return func(o *clientOptions) {
		o.decoder = decoder
	}
}

// WithErrorDecoder with client error decoder.
func WithErrorDecoder(errorDecoder DecodeErrorFunc) ClientOption {
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

// WithConfig with client config.
func WithConfig(cfg *anypb.Any) ClientOption {
	return func(o *clientOptions) {
		o.cfg = &v1.HTTPClient{}
		if cfg != nil && cfg.Value != nil {
			_ = anypb.UnmarshalTo(cfg, o.cfg, proto.UnmarshalOptions{Merge: true})
		}
	}
}

// Client is an HTTP client.
type Client struct {
	opts     clientOptions
	target   *Target
	resolver *resolver
	cc       *http.Client
	insecure bool
	selector selector.Selector
}

// NewClient returns an HTTP client.
func NewClient(ctx context.Context, options ...ClientOption) (*Client, error) {
	var opts []ClientOption
	opt := clientOptions{
		ctx:          ctx,
		timeout:      2000 * time.Millisecond,
		encoder:      DefaultRequestEncoder,
		decoder:      DefaultResponseDecoder,
		errorDecoder: DefaultErrorDecoder,
		transport:    http.DefaultTransport,
		subsetSize:   25,
	}
	if opt.cfg != nil && opt.cfg.GetTimeout().AsDuration() > 0 {
		opts = append(opts, WithTimeout(opt.cfg.GetTimeout().AsDuration()))
	}
	if opt.cfg.GetEndpoint() != "" {
		opts = append(opts, WithEndpoint(opt.cfg.GetEndpoint()))
	}
	opts = append(opts, options...)
	for _, o := range opts {
		o(&opt)
	}

	if opt.tlsConf != nil {
		if tr, ok := opt.transport.(*http.Transport); ok {
			tr.TLSClientConfig = opt.tlsConf
		}
	}
	insecure := opt.tlsConf == nil

	var (
		err    error
		target *Target
	)
	if opt.endpoint != "" {
		target, err = parseTarget(opt.endpoint, insecure)
		if err != nil {
			return nil, err
		}
	}

	selectorBuild := selector.GlobalSelector().Build()
	var r *resolver
	if opt.discovery != nil {
		if target.Scheme == "discovery" {
			if r, err = newResolver(ctx, opt.discovery, target, selectorBuild, opt.block, insecure, opt.subsetSize); err != nil {
				return nil, fmt.Errorf("[http client] new resolver failed!err: %v", opt.endpoint)
			}
		} else if _, _, err = host.ExtractHostPort(opt.endpoint); err != nil {
			return nil, fmt.Errorf("[http client] invalid endpoint format: %v", opt.endpoint)
		}
	}

	client := &Client{
		opts:     opt,
		target:   target,
		insecure: insecure,
		resolver: r,
		cc: &http.Client{
			Timeout:   opt.timeout,
			Transport: opt.transport,
		},
		selector: selectorBuild,
	}
	client.buildMiddlewareChain()

	return client, nil
}

// buildMiddlewareChain builds the middleware chain.
func (client *Client) buildMiddlewareChain() {
	clientMiddleware := client.buildClientMiddleware()
	if len(clientMiddleware) > 0 {
		userMs := client.opts.middleware
		client.opts.middleware = append(clientMiddleware, userMs...)
	}
}

// buildClientMiddleware builds the client middlewares.
func (client *Client) buildClientMiddleware() (ms []middleware.Middleware) {
	ms, _ = chain.BuildMiddleware("http.client", client.opts.cfg.GetMiddlewares())
	return ms
}

// Invoke makes a rpc call procedure for remote service.
func (client *Client) Invoke(ctx context.Context, method, path string, args interface{}, reply interface{}, opts ...CallOption) error {
	var (
		contentType string
		body        io.Reader
	)
	c := defaultCallInfo(path)
	for _, o := range opts {
		if err := o.before(&c); err != nil {
			return err
		}
	}
	if args != nil {
		data, err := client.opts.encoder(ctx, c.contentType, args)
		if err != nil {
			return err
		}
		contentType = c.contentType
		body = bytes.NewReader(data)
	}

	var url = path
	if client.target != nil {
		url = fmt.Sprintf("%s://%s%s", client.target.Scheme, client.target.Authority, path)
	}
	if c.url != nil {
		url = c.url.String()
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return err
	}
	if c.headerCarrier != nil {
		req.Header = *c.headerCarrier
	}

	if contentType != "" {
		req.Header.Set("Content-Type", c.contentType)
	}
	if client.opts.userAgent != "" {
		req.Header.Set("User-Agent", client.opts.userAgent)
	}
	app, ok := kratos.FromContext(ctx)
	if ok {
		req.Header.Set("x-md-local-caller", app.Name())
	}
	ctx = transport.NewClientContext(ctx, &Transport{
		endpoint:     client.opts.endpoint,
		reqHeader:    headerCarrier(req.Header),
		operation:    c.operation,
		request:      req,
		pathTemplate: c.pathTemplate,
	})
	return client.invoke(ctx, req, args, reply, c, opts...)
}

func (client *Client) invoke(ctx context.Context, req *http.Request, args interface{}, reply interface{}, c callInfo, opts ...CallOption) error {
	h := func(ctx context.Context, in interface{}) (interface{}, error) {
		res, err := client.do(req.WithContext(ctx))
		if res != nil {
			cs := csAttempt{res: res}
			for _, o := range opts {
				o.after(&c, &cs)
			}
		}
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		if err := client.opts.decoder(ctx, res, reply); err != nil {
			return nil, err
		}
		return reply, nil
	}
	var p selector.Peer
	ctx = selector.NewPeerContext(ctx, &p)
	if len(client.opts.middleware) > 0 {
		h = middleware.Chain(client.opts.middleware...)(h)
	}
	_, err := h(ctx, args)
	return err
}

// Do send an HTTP request and decodes the body of response into target.
// returns an error (of type *Error) if the response status code is not 2xx.
func (client *Client) Do(req *http.Request, opts ...CallOption) (*http.Response, error) {
	c := defaultCallInfo(req.URL.Path)
	for _, o := range opts {
		if err := o.before(&c); err != nil {
			return nil, err
		}
	}

	return client.do(req)
}

func (client *Client) do(req *http.Request) (*http.Response, error) {
	var done func(context.Context, selector.DoneInfo)
	// if resolver is not nil, use resolver to select node
	if client.resolver != nil {
		var (
			err  error
			node selector.Node
		)
		if node, done, err = client.selector.Select(req.Context(), selector.WithNodeFilter(client.opts.nodeFilters...)); err != nil {
			return nil, errors.ServiceUnavailable("NODE_NOT_FOUND", err.Error())
		}
		if client.insecure {
			req.URL.Scheme = "http"
		} else {
			req.URL.Scheme = "https"
		}
		req.URL.Host = node.Address()
		req.Host = node.Address()
	}
	resp, err := client.cc.Do(req)
	if err == nil {
		err = client.opts.errorDecoder(req.Context(), resp)
	}
	if done != nil {
		done(req.Context(), selector.DoneInfo{Err: err})
	}
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Close tears down the Transport and all underlying connections.
func (client *Client) Close() error {
	if client.resolver != nil {
		return client.resolver.Close()
	}
	return nil
}

// DefaultRequestEncoder is an HTTP request encoder.
func DefaultRequestEncoder(_ context.Context, contentType string, in interface{}) ([]byte, error) {
	name := httputil.ContentSubtype(contentType)
	body, err := encoding.GetCodec(name).Marshal(in)
	if err != nil {
		return nil, err
	}
	return body, err
}

// DefaultResponseDecoder is an HTTP response decoder.
func DefaultResponseDecoder(_ context.Context, res *http.Response, v interface{}) error {
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	return CodecForResponse(res).Unmarshal(data, v)
}

// DefaultErrorDecoder is an HTTP error decoder.
func DefaultErrorDecoder(_ context.Context, res *http.Response) error {
	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		return nil
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err == nil {
		e := new(errors.Error)
		if err = CodecForResponse(res).Unmarshal(data, e); err == nil {
			e.Code = int32(res.StatusCode)
			return e
		}
	}
	return errors.Newf(res.StatusCode, errors.UnknownReason, "").WithCause(err)
}

// CodecForResponse get encoding.Codec via http.Response
func CodecForResponse(r *http.Response) encoding.Codec {
	codec := encoding.GetCodec(httputil.ContentSubtype(r.Header.Get("Content-Type")))
	if codec != nil {
		return codec
	}
	return encoding.GetCodec("json")
}

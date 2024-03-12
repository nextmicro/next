package metadata

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	config "github.com/nextmicro/next/api/config/v1"
	v1 "github.com/nextmicro/next/api/middleware/metadata/v1"
	chain "github.com/nextmicro/next/middleware"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const namespace = "metadata"

func init() {
	chain.Register("client."+namespace, injectionClient)
	chain.Register("server."+namespace, injectionServer)
}

// Option is metadata option.
type Option func(*options)

// // Option is metadata option.
// type Option func(*options)
type options struct {
	prefix []string
	md     metadata.Metadata
}

func (o *options) hasPrefix(key string) bool {
	k := strings.ToLower(key)
	for _, prefix := range o.prefix {
		if strings.HasPrefix(k, prefix) {
			return true
		}
	}
	return false
}

// WithConstants with constant metadata key value.
func WithConstants(md metadata.Metadata) Option {
	return func(o *options) {
		o.md = md
	}
}

// WithPropagatedPrefix with propagated key prefix.
func WithPropagatedPrefix(prefix ...string) Option {
	return func(o *options) {
		o.prefix = prefix
	}
}

func injectionServer(c *config.Middleware) (middleware.Middleware, error) {
	cfg := &v1.Metadata{
		Prefix: []string{"x-md-"}, // x-md-global-, x-md-local
	}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, cfg, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}

	opts := make([]Option, 0, 1)
	if len(cfg.Prefix) > 0 {
		opts = append(opts, WithPropagatedPrefix(cfg.Prefix...))
	}

	return Server(opts...), nil
}

// Server is middleware server-side metadata.
func Server(opts ...Option) middleware.Middleware {
	options := &options{
		prefix: []string{"x-md-"}, // x-md-global-, x-md-local
	}
	for _, o := range opts {
		o(options)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			md := options.md.Clone()
			header := tr.RequestHeader()
			for _, k := range header.Keys() {
				if options.hasPrefix(k) {
					for _, v := range header.Values(k) {
						md.Add(k, v)
					}
				}
			}
			ctx = metadata.NewServerContext(ctx, md)
			return handler(ctx, req)
		}
	}
}

func injectionClient(c *config.Middleware) (middleware.Middleware, error) {
	cfg := &v1.Metadata{
		Prefix: []string{"x-md-global-"},
	}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, cfg, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}

	opts := make([]Option, 0, 1)
	if len(cfg.Prefix) > 0 {
		opts = append(opts, WithPropagatedPrefix(cfg.Prefix...))
	}

	return Client(opts...), nil
}

// Client is middleware client-side metadata.
func Client(opts ...Option) middleware.Middleware {
	options := &options{
		prefix: []string{"x-md-global-"},
	}
	for _, o := range opts {
		o(options)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			tr, ok := transport.FromClientContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			header := tr.RequestHeader()
			// x-md-local-
			for k, vList := range options.md {
				for _, v := range vList {
					header.Add(k, v)
				}
			}
			if md, ok := metadata.FromClientContext(ctx); ok {
				for k, vList := range md {
					for _, v := range vList {
						header.Add(k, v)
					}
				}
			}
			// x-md-global-
			if md, ok := metadata.FromServerContext(ctx); ok {
				for k, vList := range md {
					if options.hasPrefix(k) {
						for _, v := range vList {
							header.Add(k, v)
						}
					}
				}
			}
			return handler(ctx, req)
		}
	}
}

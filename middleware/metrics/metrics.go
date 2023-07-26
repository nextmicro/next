package metrics

import (
	"context"
	"time"

	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/metrics"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/transport"
	config "github.com/nextmicro/next/api/config/v1"
	v1 "github.com/nextmicro/next/api/middleware/metrics/v1"
	chain "github.com/nextmicro/next/middleware"
	metric "github.com/nextmicro/next/pkg/metrics"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const namespace = "metrics"

func init() {
	chain.Register("client."+namespace, Client)
	chain.Register("server."+namespace, Server)
}

type Option func(o *Options)

type Options struct {
	*v1.Metrics

	// counter: <client/server>_requests_code_total{kind, operation, code, reason}
	requests metrics.Counter

	// histogram: <client/server>_requests_seconds_bucket{kind, operation}
	seconds metrics.Observer
}

// Client is middleware client-side metrics.
func Client(c *config.Middleware) (middleware.Middleware, error) {
	options := Options{
		requests: prom.NewCounter(metric.ClientMetricRequests),
		seconds:  prom.NewHistogram(metric.ClientMetricMillisecond),
	}

	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, options, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			var (
				kind   string
				method string
				status string
				callee = "unknown"
			)

			startTime := time.Now()
			if peer, ok := selector.FromPeerContext(ctx); ok && peer.Node != nil {
				callee = peer.Node.ServiceName()
			}

			if info, ok := transport.FromClientContext(ctx); ok {
				kind = info.Kind().String()
				method = info.Operation()
			}
			reply, err := handler(ctx, req)
			status = metric.FromErrorCode(err).String()
			if options.requests != nil {
				options.requests.With(kind, callee, method, status).Inc()
			}
			if options.seconds != nil {
				options.seconds.With(kind, callee, method).Observe(float64(time.Since(startTime).Milliseconds()))
			}

			return reply, err
		}
	}, nil
}

// Server wraps a server.Server with prometheus metrics.
func Server(c *config.Middleware) (middleware.Middleware, error) {
	options := Options{
		requests: prom.NewCounter(metric.ClientMetricRequests),
		seconds:  prom.NewHistogram(metric.ClientMetricMillisecond),
	}

	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, options, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {

			var (
				kind   string
				method string
				caller string
				status string
			)
			startTime := time.Now()
			if md, ok := metadata.FromServerContext(ctx); ok {
				if v := md.Get("x-md-local-caller"); v != "" {
					caller = v
				}
			}
			if info, ok := transport.FromServerContext(ctx); ok {
				kind = info.Kind().String()
				method = info.Operation()
			}
			reply, err := handler(ctx, req)
			status = metric.FromErrorCode(err).String()
			if options.requests != nil {
				options.requests.With(kind, caller, method, status).Inc()
			}
			if options.seconds != nil {
				options.seconds.With(kind, caller, method).Observe(float64(time.Since(startTime).Milliseconds()))
			}

			return reply, err
		}
	}, nil
}

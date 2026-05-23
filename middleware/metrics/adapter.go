package metrics

import (
	"context"
	"time"

	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/transport"
	metric "github.com/nextmicro/next/pkg/metrics"
)

// Client is middleware client-side metrics (next Prometheus 命名约定 + kratos contrib 后端).
func Client(opts ...Option) middleware.Middleware {
	options := Options{
		requests: prom.NewCounter(metric.ClientMetricRequests),
		seconds:  prom.NewHistogram(metric.ClientMetricMillisecond),
	}
	for _, o := range opts {
		o(&options)
	}
	if options.disabled {
		return func(handler middleware.Handler) middleware.Handler { return handler }
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
			if info, ok := transport.FromClientContext(ctx); ok {
				kind = info.Kind().String()
				method = info.Operation()
			}
			reply, err := handler(ctx, req)
			if peer, ok := selector.FromPeerContext(ctx); ok && peer.Node != nil {
				callee = peer.Node.ServiceName()
			}
			status = metric.FromErrorCode(err).String()
			if options.requests != nil {
				options.requests.With(kind, callee, method, status).Inc()
			}
			if options.seconds != nil {
				options.seconds.With(kind, callee, method).Observe(float64(time.Since(startTime).Milliseconds()))
			}
			return reply, err
		}
	}
}

// Server wraps a server with next Prometheus metrics.
func Server(opts ...Option) middleware.Middleware {
	options := Options{
		requests: prom.NewCounter(metric.ServerMetricRequests),
		seconds:  prom.NewHistogram(metric.ServerMetricMillisecond),
	}
	for _, o := range opts {
		o(&options)
	}
	if options.disabled {
		return func(handler middleware.Handler) middleware.Handler { return handler }
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
	}
}

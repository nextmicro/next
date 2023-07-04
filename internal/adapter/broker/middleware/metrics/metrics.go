package metrics

import (
	"context"
	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	"github.com/go-kratos/kratos/v2/metrics"
	"github.com/nextmicro/next/internal/adapter/broker/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/codes"
	"time"
)

var (
	ComponentNamespace = "component"

	// MessagingProducerMetricMillisecond is a prometheus histogram for measuring the duration of a request.
	MessagingProducerMetricMillisecond = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: ComponentNamespace,
		Subsystem: "messaging_producer_requests",
		Name:      "duration_ms",
		Help:      "requests duration(ms).",
		Buckets:   []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
	}, []string{"kind", "addr", "destination"})

	// MessagingProducerMetricRequests  is a counter vector of requests.
	MessagingProducerMetricRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ComponentNamespace,
		Subsystem: "messaging_producer_requests",
		Name:      "total",
		Help:      "The total number of processed requests",
	}, []string{"kind", "addr", "destination", "status"})

	// MessagingConsumerMetricMillisecond is a prometheus histogram for measuring the duration of a request.
	MessagingConsumerMetricMillisecond = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: ComponentNamespace,
		Subsystem: "messaging_consumer_requests",
		Name:      "duration_ms",
		Help:      "requests duration(ms).",
		Buckets:   []float64{5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
	}, []string{"kind", "addr", "destination", "group"})

	// MessagingConsumerMetricRequests  is a counter vector of requests.
	MessagingConsumerMetricRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: ComponentNamespace,
		Subsystem: "messaging_consumer_requests",
		Name:      "total",
		Help:      "The total number of processed requests",
	}, []string{"kind", "addr", "destination", "group", "status"})
)

// Client  metrics.
func Client(opts ...Option) middleware.Middleware {
	op := &options{
		requests:    prom.NewCounter(MessagingProducerMetricRequests),
		millisecond: prom.NewHistogram(MessagingProducerMetricMillisecond),
	}
	for _, opt := range opts {
		opt(op)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, topic string, req interface{}) (reply interface{}, err error) {
			start := time.Now()

			reply, err = handler(ctx, topic, req)
			if err != nil {
				return
			}

			var code = codes.Ok
			if err != nil {
				code = codes.Error
			}

			op.requests.With(op.namespace, op.addr, topic, code.String()).Inc()
			op.millisecond.With(op.namespace, op.addr, topic).Observe(float64(time.Since(start).Milliseconds()))

			return
		}
	}
}

// Server  metrics.
func Server(opts ...Option) middleware.Middleware {
	op := &options{
		requests:    prom.NewCounter(metrics.MessagingConsumerMetricRequests),
		millisecond: prom.NewHistogram(metrics.MessagingConsumerMetricMillisecond),
	}
	for _, opt := range opts {
		opt(op)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, topic string, req interface{}) (reply interface{}, err error) {
			start := time.Now()

			reply, err = handler(ctx, topic, req)
			if err != nil {
				return
			}

			var code = codes.Ok
			if err != nil {
				code = codes.Error
			}

			op.requests.With(op.namespace, op.addr, topic, op.group, code.String()).Inc()
			op.millisecond.With(op.namespace, op.addr, topic, op.group).Observe(float64(time.Since(start).Milliseconds()))

			return
		}
	}
}

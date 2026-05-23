package metrics

import (
	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	"github.com/go-kratos/kratos/v2/metrics"
	metric "github.com/nextmicro/next/pkg/metrics"
)

type Option func(o *Options)

type Options struct {
	disabled bool
	requests metrics.Counter
	seconds  metrics.Observer
}

// WithDisabled set disabled metrics.
func WithDisabled(disabled bool) Option {
	return func(o *Options) {
		o.disabled = disabled
	}
}

// WithRequests with requests counter.
func WithRequests(c metrics.Counter) Option {
	return func(o *Options) {
		o.requests = c
	}
}

// WithSeconds with seconds histogram.
func WithSeconds(c metrics.Observer) Option {
	return func(o *Options) {
		o.seconds = c
	}
}

func defaultClientOptions() []Option {
	return []Option{
		WithRequests(prom.NewCounter(metric.ClientMetricRequests)),
		WithSeconds(prom.NewHistogram(metric.ClientMetricMillisecond)),
	}
}

func defaultServerOptions() []Option {
	return []Option{
		WithRequests(prom.NewCounter(metric.ServerMetricRequests)),
		WithSeconds(prom.NewHistogram(metric.ServerMetricMillisecond)),
	}
}

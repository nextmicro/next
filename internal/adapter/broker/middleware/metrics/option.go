package metrics

import (
	"github.com/go-kratos/kratos/v2/metrics"
)

// Option is metrics option.
type Option func(options *options)

type options struct {
	disabled    bool
	namespace   string
	addr        string
	group       string
	requests    metrics.Counter
	millisecond metrics.Observer
}

// WithDisabled set disabled metrics.
func WithDisabled(disabled bool) Option {
	return func(o *options) {
		o.disabled = disabled
	}
}

func WithNamespace(namespace string) Option {
	return func(o *options) {
		o.namespace = namespace
	}
}

func WithGroup(group string) Option {
	return func(o *options) {
		o.group = group
	}
}

// WithAddr with addr label.
func WithAddr(address string) Option {
	return func(o *options) {
		o.addr = address
	}
}

// WithRequests with requests counter.
func WithRequests(c metrics.Counter) Option {
	return func(o *options) {
		o.requests = c
	}
}

// WithMillisecond with seconds histogram.
func WithMillisecond(c metrics.Observer) Option {
	return func(o *options) {
		o.millisecond = c
	}
}

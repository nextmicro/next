package tracing

import (
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*options)
}

type optionFunc func(*options)

func (o optionFunc) apply(c *options) {
	o(c)
}

type options struct {
	disabled       bool
	tracerProvider oteltrace.TracerProvider
	propagators    propagation.TextMapPropagator
}

// WithDisabled set disabled trace.
func WithDisabled(disabled bool) Option {
	return optionFunc(func(o *options) {
		o.disabled = disabled
	})
}

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global provider is used.
func WithTracerProvider(provider oteltrace.TracerProvider) Option {
	return optionFunc(func(o *options) {
		if provider != nil {
			o.tracerProvider = provider
		}
	})
}

// WithPropagators specifies propagators to use for extracting
// information from the HTTP requests. If none are specified, global
// ones will be used.
func WithPropagators(propagators propagation.TextMapPropagator) Option {
	return optionFunc(func(o *options) {
		if propagators != nil {
			o.propagators = propagators
		}
	})
}

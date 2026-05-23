package tracing

import (
	"go.opentelemetry.io/contrib/propagators/b3"
	ktracing "github.com/go-kratos/kratos/v2/middleware/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Option specifies instrumentation configuration options.
type Option interface {
	apply(*options)
}

type optionFunc func(*options)

func (o optionFunc) apply(c *options) { o(c) }

type options struct {
	disabled       bool
	tracerProvider oteltrace.TracerProvider
	propagators    propagation.TextMapPropagator
}

func defaultOptions() options {
	return options{
		tracerProvider: otel.GetTracerProvider(),
		propagators:    DefaultPropagator(),
	}
}

// DefaultPropagator returns the next default composite propagator (b3 + metadata + w3c).
func DefaultPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		ktracing.Metadata{},
		propagation.Baggage{},
		propagation.TraceContext{},
		b3.New(),
	)
}

// WithDisabled disables tracing middleware.
func WithDisabled(disabled bool) Option {
	return optionFunc(func(o *options) {
		o.disabled = disabled
	})
}

// WithTracerProvider specifies a tracer provider.
func WithTracerProvider(provider oteltrace.TracerProvider) Option {
	return optionFunc(func(o *options) {
		if provider != nil {
			o.tracerProvider = provider
		}
	})
}

// WithPropagators specifies propagators for extract/inject.
func WithPropagators(propagators propagation.TextMapPropagator) Option {
	return optionFunc(func(o *options) {
		if propagators != nil {
			o.propagators = propagators
		}
	})
}

// WithB3Propagator ensures b3 is included in the default composite propagator.
func WithB3Propagator() Option {
	return optionFunc(func(o *options) {
		o.propagators = DefaultPropagator()
	})
}

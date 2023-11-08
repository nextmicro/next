package tracing

import (
	"context"

	"github.com/nextmicro/next/runtime/loader"
	"go.opentelemetry.io/otel/attribute"
)

type tracingKey struct{}

type Config struct {
	Endpoint   string
	Sampler    float64
	Batcher    string
	Attributes []attribute.KeyValue
}

// WithConfig sets the logger config
func WithConfig(cfg Config) loader.Option {
	return func(o *loader.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, tracingKey{}, cfg)
	}
}

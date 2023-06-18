package tracing

import (
	"context"

	tr "github.com/nextmicro/gokit/trace"
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

func options(c Config) []tr.Option {
	var opts []tr.Option
	if c.Endpoint != "" {
		opts = append(opts, tr.WithEndpoint(c.Endpoint))
	}
	if c.Sampler != 0 {
		opts = append(opts, tr.WithSampler(c.Sampler))
	}
	if c.Batcher != "" {
		opts = append(opts, tr.WithBatcher(c.Batcher))
	}
	opts = append(opts, tr.WithAttributes(c.Attributes...))
	return opts
}

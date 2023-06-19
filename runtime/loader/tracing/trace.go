package tracing

import (
	"context"

	"github.com/go-volo/logger"
	tr "github.com/nextmicro/gokit/trace"
	"github.com/nextmicro/next/api/config/v1"
	"github.com/nextmicro/next/config"
	"github.com/nextmicro/next/runtime/loader"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

type Tracing struct {
	initialized bool
	provider    *tr.Tracing
	opt         loader.Options
}

func New(opts ...loader.Option) loader.Loader {
	o := loader.Options{}
	for _, opt := range opts {
		opt(&o)
	}

	return &Tracing{
		opt: o,
	}
}

func (loader *Tracing) Init(...loader.Option) (err error) {
	var cfg = config.AppConfig().GetTracing()
	if cfg == nil {
		cfg = &v1.Tracing{
			Disable: true,
		}
	}
	if cfg.Disable {
		return nil
	}

	var opts = []tr.Option{
		tr.WithEndpoint(cfg.Endpoint),
		tr.WithSampler(cfg.Sampler),
		tr.WithAttributes(
			attribute.String("service.id", config.AppConfig().GetId()),
			semconv.ServiceNameKey.String(config.AppConfig().GetName()),
			semconv.ServiceVersionKey.String(config.AppConfig().GetVersion()),
		),
	}

	loader.provider, err = tr.New(opts...)
	if err != nil {
		return err
	}

	loader.initialized = true
	logger.Infof("Loader [%s] Init success", loader.String())

	return nil
}

func (loader *Tracing) Start(ctx context.Context) error {
	return nil
}

func (loader *Tracing) Watch() error {
	return nil
}

func (loader *Tracing) Stop(ctx context.Context) error {
	if !loader.initialized {
		return nil
	}
	if err := loader.provider.Shutdown(ctx); err != nil {
		return err
	}

	logger.Infof("Loader [%s] Stop success", loader.String())
	return nil
}

func (loader *Tracing) String() string {
	return "otel"
}

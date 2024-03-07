package tracing

import (
	"context"

	tr "github.com/nextmicro/gokit/trace"
	"github.com/nextmicro/logger"
	"github.com/nextmicro/next/config"
	"github.com/nextmicro/next/pkg/env"
	"github.com/nextmicro/next/runtime/loader"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

type Tracing struct {
	provider *tr.Tracing
	opt      loader.Options
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

func (loader *Tracing) Initialized() bool {
	return loader.opt.Initialized
}

func (loader *Tracing) Init(...loader.Option) (err error) {
	var cfg = config.ApplicationConfig().GetTelemetry()
	if cfg == nil || cfg.Disable {
		return nil
	}

	var opts = []tr.Option{
		tr.WithEndpoint(cfg.Endpoint),
		tr.WithBatcher(cfg.Exporter),
		tr.WithSampler(cfg.Sampler),
		tr.WithOtlpHeaders(cfg.GetHeaders()),
		tr.WithOtlpHttpPath(cfg.GetHttpPath()),
		tr.WithAttributes(
			attribute.String("service.id", config.ApplicationConfig().GetId()),
			semconv.ServiceName(config.ApplicationConfig().GetName()),
			semconv.ServiceVersion(config.ApplicationConfig().GetVersion()),
			semconv.ServiceInstanceID(env.Hostname()),
			semconv.DeploymentEnvironment(env.DeployEnvironment()),
		),
	}

	loader.provider, err = tr.New(opts...)
	if err != nil {
		return errors.WithStack(err)
	}

	loader.opt.Initialized = true

	return nil
}

func (loader *Tracing) Start(ctx context.Context) error {
	logger.Infof("OTEL [%s] Start success", config.ApplicationConfig().GetTelemetry().Exporter)
	return nil
}

func (loader *Tracing) Watch() error {
	return nil
}

func (loader *Tracing) Stop(ctx context.Context) error {
	if err := loader.provider.Shutdown(ctx); err != nil {
		logger.Errorf("OTEL [%s] Stop error: %s", config.ApplicationConfig().GetTelemetry().Exporter, err.Error())
		return errors.WithStack(err)
	}

	logger.Infof("OTEL [%s] Stop success", config.ApplicationConfig().GetTelemetry().Exporter)
	return nil
}

func (loader *Tracing) String() string {
	return "otel"
}

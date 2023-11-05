package tracing

import (
	"context"

	tr "github.com/nextmicro/gokit/trace"
	"github.com/nextmicro/logger"
	"github.com/nextmicro/next/api/config/v1"
	"github.com/nextmicro/next/config"
	"github.com/nextmicro/next/pkg/env"
	"github.com/nextmicro/next/runtime/loader"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
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
	var cfg = config.ApplicationConfig().GetTelemetry()
	if cfg == nil {
		cfg = &v1.Telemetry{}
	}
	if cfg.Disable {
		return nil
	}
	var exporter = tr.KindStdout
	if cfg.Exporter != "" {
		exporter = cfg.Exporter
	}

	var opts = []tr.Option{
		tr.WithEndpoint(cfg.Endpoint),
		tr.WithBatcher(exporter),
		tr.WithSampler(cfg.Sampler),
		tr.WithOtlpHeaders(cfg.GetOTLPHeaders()),
		tr.WithOtlpHttpPath(cfg.GetOPLPHttpPath()),
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
	return "OTEL"
}

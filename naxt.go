package next

import (
	"context"
	"os"
	"syscall"
	"time"

	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	v1 "github.com/nextmicro/next/api/config/v1"
	"github.com/nextmicro/next/config"
	"github.com/nextmicro/next/pkg/env"
	metric "github.com/nextmicro/next/pkg/metrics"
	"github.com/nextmicro/next/registry"

	"github.com/go-kratos/kratos/v2"
	_ "github.com/nextmicro/next/middleware/bbr"
	_ "github.com/nextmicro/next/middleware/circuitbreaker"
	_ "github.com/nextmicro/next/middleware/logging"
	_ "github.com/nextmicro/next/middleware/metadata"
	_ "github.com/nextmicro/next/middleware/metrics"
	_ "github.com/nextmicro/next/middleware/recovery"
	_ "github.com/nextmicro/next/middleware/tracing"
	"github.com/nextmicro/next/runtime"
)

type Next struct {
	*kratos.App
	opt Options
}

// New create an application lifecycle manager.
func New(opts ...Option) (*Next, error) {
	opt := buildOptions(config.ApplicationConfig(), opts...)

	run := runtime.NewRuntime()
	if err := run.Init(
		runtime.ID(opt.ID),
		runtime.Name(opt.Name),
		runtime.Version(opt.Version),
		runtime.Metadata(opt.Metadata),
		runtime.Loader(opt.Loader...),
	); err != nil {
		return nil, err
	}

	if err := run.Start(opt.Ctx); err != nil {
		return nil, err
	}

	opt.AfterStop = append(opt.AfterStop, run.Stop)

	kOpts := []kratos.Option{
		kratos.ID(opt.ID),
		kratos.Name(opt.Name),
		kratos.Version(opt.Version),
		kratos.Metadata(opt.Metadata),
		kratos.Endpoint(opt.Endpoints...),
		kratos.Context(opt.Ctx),
		kratos.Server(opt.Servers...),
		kratos.Signal(opt.Sigs...),
		kratos.Registrar(opt.Registrar),
		kratos.RegistrarTimeout(opt.RegistrarTimeout),
		kratos.StopTimeout(opt.StopTimeout),
	}
	for _, beforeStart := range opt.BeforeStart {
		kOpts = append(kOpts, kratos.BeforeStart(beforeStart))
	}
	for _, beforeStop := range opt.BeforeStop {
		kOpts = append(kOpts, kratos.BeforeStop(beforeStop))
	}
	for _, afterStart := range opt.AfterStart {
		kOpts = append(kOpts, kratos.AfterStart(afterStart))
	}
	for _, afterStop := range opt.AfterStop {
		kOpts = append(kOpts, kratos.AfterStop(afterStop))
	}

	return &Next{
		App: kratos.New(kOpts...),
		opt: opt,
	}, nil
}

// buildOptions build options
func buildOptions(cfg *v1.Next, opts ...Option) Options {
	opt := Options{
		Ctx:              context.Background(),
		Sigs:             []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL},
		Registrar:        registry.DefaultRegistry,
		RegistrarTimeout: 10 * time.Second,
		StopTimeout:      10 * time.Second,
	}
	for _, o := range opts {
		o(&opt)
	}

	if cfg != nil {
		if opt.ID == "" && cfg.GetId() != "" {
			opt.ID = cfg.GetId()
		}
		if opt.Name == "" && cfg.GetName() != "" {
			opt.Name = cfg.GetName()
		}
		if opt.Version == "" && cfg.GetVersion() != "" {
			opt.Version = cfg.GetVersion()
		}
		if opt.Metadata == nil && cfg.GetMetadata() != nil {
			opt.Metadata = cfg.GetMetadata()
		}
	}

	if cfg.GetId() == "" && opt.ID != "" {
		cfg.Id = opt.ID
	}
	if cfg.GetName() == "" && opt.Name != "" {
		cfg.Name = opt.Name
	}
	if cfg.GetVersion() == "" && opt.Version != "" {
		cfg.Version = opt.Version
	}
	if cfg.GetMetadata() == nil && opt.Metadata != nil {
		cfg.Metadata = opt.Metadata
	}

	// build app metrics.
	prom.NewGauge(metric.BuildInfoGauge).With(
		cfg.GetId(),
		cfg.GetName(),
		cfg.GetVersion(),
		env.DeployEnvironment(),
		env.GoVersion(),
		env.AppVersion(),
		env.StartTime(),
		env.BuildTime(),
	).Set(float64(time.Now().UnixNano() / 1e6))

	return opt
}

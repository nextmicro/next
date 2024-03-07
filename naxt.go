package next

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
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
	opt := buildOptions(opts...)

	// register runtime
	run := runtime.NewRuntime()
	if err := run.Init(
		runtime.Loader(opt.Loader...),
	); err != nil {
		return nil, fmt.Errorf("runtime init error: %w", err)
	}

	// start runtime
	if err := run.Start(opt.Ctx); err != nil {
		return nil, fmt.Errorf("runtime start error: %w", err)
	}

	// register runtime stop
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

	next := &Next{
		App: kratos.New(kOpts...),
		opt: opt,
	}

	// init metrics
	next.initMetrics()

	return next, nil
}

func (app *Next) initMetrics() {
	// build app metrics.
	prom.NewGauge(metric.BuildInfoGauge).With(
		app.ID(),
		app.Name(),
		app.Version(),
		env.DeployEnvironment(),
		env.GoVersion(),
		env.AppVersion(),
		env.StartTime(),
		env.BuildTime(),
	).Set(float64(time.Now().UnixNano() / 1e6))
}

// buildOptions build options
func buildOptions(options ...Option) Options {
	var opts []Option
	opt := Options{
		Ctx:              context.Background(),
		Sigs:             []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL},
		Registrar:        registry.DefaultRegistry,
		RegistrarTimeout: 10 * time.Second,
		StopTimeout:      10 * time.Second,
	}

	c := config.ApplicationConfig()
	if c.GetId() != "" {
		opts = append(opts, ID(c.GetId()))
	}
	if c.GetName() != "" {
		opts = append(opts, Name(c.GetName()))
	}
	if c.GetVersion() != "" {
		opts = append(opts, Version(c.GetVersion()))
	}
	if c.GetMetadata() != nil {
		opts = append(opts, Metadata(c.GetMetadata()))
	}
	opts = append(opts, options...)

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

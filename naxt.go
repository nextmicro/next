package next

import (
	"context"
	"os"
	"syscall"
	"time"

	"github.com/go-kratos/kratos/v2"
	"github.com/nextmicro/next/runtime"
)

type Next struct {
	*kratos.App
	opt Options
}

// New create an application lifecycle manager.
func New(opts ...Option) (*Next, error) {
	opt := Options{
		Ctx:              context.Background(),
		Sigs:             []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL},
		RegistrarTimeout: 10 * time.Second,
		StopTimeout:      10 * time.Second,
	}
	for _, o := range opts {
		o(&opt)
	}

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

	opt.BeforeStart = append(opt.BeforeStart, run.Start)
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
	}, nil
}

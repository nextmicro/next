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
		ctx:              context.Background(),
		sigs:             []os.Signal{syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL},
		registrarTimeout: 10 * time.Second,
		stopTimeout:      10 * time.Second,
	}
	for _, o := range opts {
		o(&opt)
	}

	run := runtime.NewRuntime()
	if err := run.Init(); err != nil {
		return nil, err
	}

	opt.beforeStart = append(opt.beforeStart, run.Start)
	opt.afterStop = append(opt.afterStop, run.Stop)

	kOpts := []kratos.Option{
		kratos.ID(opt.id),
		kratos.Name(opt.name),
		kratos.Version(opt.version),
		kratos.Metadata(opt.metadata),
		kratos.Endpoint(opt.endpoints...),
		kratos.Context(opt.ctx),
		kratos.Server(opt.servers...),
		kratos.Signal(opt.sigs...),
		kratos.Registrar(opt.registrar),
		kratos.RegistrarTimeout(opt.registrarTimeout),
		kratos.StopTimeout(opt.stopTimeout),
	}
	for _, beforeStart := range opt.beforeStart {
		kOpts = append(kOpts, kratos.BeforeStart(beforeStart))
	}
	for _, beforeStop := range opt.beforeStop {
		kOpts = append(kOpts, kratos.BeforeStop(beforeStop))
	}
	for _, afterStart := range opt.afterStart {
		kOpts = append(kOpts, kratos.AfterStart(afterStart))
	}
	for _, afterStop := range opt.afterStop {
		kOpts = append(kOpts, kratos.AfterStop(afterStop))
	}

	return &Next{
		App: kratos.New(kOpts...),
	}, nil
}

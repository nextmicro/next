package runtime

import (
	"context"
	"errors"
)

type Runtime struct {
	// options configure runtime
	options Options
}

// NewRuntime creates new local runtime and returns it
func NewRuntime(opts ...Option) *Runtime {
	// get default options
	options := applyOptions(opts...)
	return &Runtime{
		options: options,
	}
}

// Init initializes runtime
func (r *Runtime) Init(opts ...Option) error {
	for _, o := range opts {
		o(&r.options)
	}

	for _, load := range r.options.loader {
		if err := load.Init(); err != nil {
			return errors.New(load.String() + ": init failed " + err.Error())
		}
	}

	return nil
}

// Start runtime start
func (r *Runtime) Start(ctx context.Context) error {
	for _, load := range r.options.loader {
		if !load.Initialized() {
			continue
		}
		err := load.Start(ctx)
		if err != nil {
			return errors.New(load.String() + ": start failed " + err.Error())
		}

		err = load.Watch()
		if err != nil {
			return errors.New(load.String() + ": watch failed " + err.Error())
		}
	}

	return nil
}

// Stop stops runtime
func (r *Runtime) Stop(ctx context.Context) (err error) {
	// reverse stop
	for i := len(r.options.loader) - 1; i >= 0; i-- {
		load := r.options.loader[i]
		if !load.Initialized() {
			continue
		}
		if err = load.Stop(ctx); err != nil && !errors.Is(err, context.Canceled) {
			err = errors.New(load.String() + ": stop failed " + err.Error())
		}
	}

	return err
}

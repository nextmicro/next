package runtime

import (
	"context"
	"errors"
)

type runtime struct {
	// options configure runtime
	options Options
}

// NewRuntime creates new local runtime and returns it
func NewRuntime(opts ...Option) Runtime {
	// get default options
	options := defaultOptions(opts...)

	return &runtime{
		options: options,
	}
}

// Init initializes runtime
func (r *runtime) Init(opts ...Option) error {
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
func (r *runtime) Start(ctx context.Context) error {
	for _, load := range r.options.loader {
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
func (r *runtime) Stop(ctx context.Context) error {
	// reverse stop
	for i := len(r.options.loader) - 1; i >= 0; i-- {
		load := r.options.loader[i]
		if err := load.Stop(ctx); err != nil && !errors.Is(err, context.Canceled) {
			err = errors.New(load.String() + ": stop failed " + err.Error())
		}
	}

	return nil
}

func (r *runtime) ID() string {
	return r.options.id
}

func (r *runtime) Name() string {
	return r.options.name
}

func (r *runtime) Version() string {
	return r.options.version
}

func (r *runtime) Metadata() map[string]string {
	return r.options.metadata
}

func (r *runtime) String() string {
	return "default"
}

package loader

import (
	"context"
)

type Loader interface {
	// Initialized returns true if loader is initialized
	Initialized() bool
	// Init initializes loader
	Init(opts ...Option) error
	// Start starts loader
	Start(ctx context.Context) error
	// Watch watches for changes
	Watch() error
	// Stop stops loader
	Stop(ctx context.Context) error
	// String returns loader name
	String() string
}

type BaseLoader struct {
}

func (loader *BaseLoader) Init(opts ...Option) error {
	return nil
}

func (loader *BaseLoader) Initialized() bool {
	return true
}

func (loader *BaseLoader) Start(ctx context.Context) error {
	return nil
}

func (loader *BaseLoader) Watch() error {
	return nil
}

func (loader *BaseLoader) Stop(ctx context.Context) error {
	return nil
}

func (loader *BaseLoader) String() string {
	return ""
}

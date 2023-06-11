package loader

import (
	"context"
)

type Loader interface {
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

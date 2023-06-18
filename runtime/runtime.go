package runtime

import (
	"context"
)

// Runtime is a service runtime manager
type Runtime interface {
	// ID returns runtime ID
	ID() string
	// Name returns runtime name
	Name() string
	// Version returns runtime version
	Version() string
	// Metadata returns runtime metadata
	Metadata() map[string]string
	// Init initializes runtime
	Init(...Option) error
	// Start starts the runtime
	Start(ctx context.Context) error
	// Stop shuts down the runtime
	Stop(context.Context) error
	// String describes runtime
	String() string
}

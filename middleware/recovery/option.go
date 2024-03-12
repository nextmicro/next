package recovery

import (
	"context"

	"github.com/nextmicro/logger"
)

// HandlerFunc is recovery handler func.
type HandlerFunc func(ctx context.Context, err interface{}) error

// Option is recovery option.
type Option func(*options)

type options struct {
	// disabled recovery.
	disabled bool

	// Size of the stack to be printed.
	// Optional. Default value 4KB.
	stackSize int

	// DisableStackAll disables formatting stack traces of all other goroutines
	// into buffer after the trace for the current goroutine.
	// Optional. Default value false.
	disableStackAll bool

	// DisablePrintStack disables printing stack trace.
	// Optional. Default value as false.
	disablePrintStack bool

	// logger is recovery logger.
	logger logger.Logger

	// handler is recovery handler.
	handler HandlerFunc
}

// WithDisabled set disabled recovery.
func WithDisabled(disabled bool) Option {
	return func(o *options) {
		o.disabled = disabled
	}
}

// WithStackSize with stack size.
func WithStackSize(size int) Option {
	return func(o *options) {
		o.stackSize = size
	}
}

// WithDisableStackAll disables formatting stack traces of all other goroutines
func WithDisableStackAll(v bool) Option {
	return func(o *options) {
		o.disableStackAll = v
	}
}

// WithDisablePrintStack DisablePrintStack disables printing stack trace.
func WithDisablePrintStack(v bool) Option {
	return func(o *options) {
		o.disablePrintStack = v
	}
}

// WithLogger with recovery logger.
func WithLogger(log logger.Logger) Option {
	return func(o *options) {
		o.logger = log
	}
}

// WithHandler with recovery handler.
func WithHandler(h HandlerFunc) Option {
	return func(o *options) {
		o.handler = h
	}
}

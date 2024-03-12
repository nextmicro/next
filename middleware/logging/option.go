package logging

import (
	"context"
	"time"

	"github.com/nextmicro/logger"
)

const (
	DefaultFormat = "2006-01-02T15:04:05.999Z0700"
)

// Option is the option for the logger.
type Option func(o *Options)

// Options is the options for the logger.
type Options struct {
	disabled      bool
	timeFormat    string
	slowThreshold time.Duration
	logger        logger.Logger
	handler       func(ctx context.Context, req any) map[string]string
}

// WithDisabled set disabled metrics.
func WithDisabled(disabled bool) Option {
	return func(o *Options) {
		o.disabled = disabled
	}
}

// WithTimeFormat sets the format of the time
func WithTimeFormat(format string) Option {
	return func(o *Options) {
		o.timeFormat = format
	}
}

// WithSlowThreshold sets the slow threshold.
func WithSlowThreshold(threshold time.Duration) Option {
	return func(o *Options) {
		o.slowThreshold = threshold
	}
}

// WithLogger sets the logger
func WithLogger(log logger.Logger) Option {
	return func(o *Options) {
		o.logger = log
	}
}

// WithHandler sets the handler
func WithHandler(handler func(ctx context.Context, req any) map[string]string) Option {
	return func(o *Options) {
		o.handler = handler
	}
}

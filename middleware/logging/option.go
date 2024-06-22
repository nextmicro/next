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
	accessLevel   logger.Level
	ignoredRoutes []string
	Metadata      []Metadata
	handler       func(ctx context.Context, req any) map[string]string
	dumpReq       bool
	dumpResp      bool
}

type Metadata struct {
	Key    string // key
	Rename string // rename
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

// WithAccessLevel sets the access level
func WithAccessLevel(level logger.Level) Option {
	return func(o *Options) {
		o.accessLevel = level
	}
}

// WithHandler sets the handler
func WithHandler(handler func(ctx context.Context, req any) map[string]string) Option {
	return func(o *Options) {
		o.handler = handler
	}
}

// WithIgnoredRoutes sets the ignored routes
func WithIgnoredRoutes(routes []string) Option {
	return func(o *Options) {
		o.ignoredRoutes = append(o.ignoredRoutes, routes...)
	}
}

// WithMetadata sets the metadata
func WithMetadata(md []Metadata) Option {
	return func(o *Options) {
		o.Metadata = append(o.Metadata, md...)
	}
}

func WithDumpRequest(dump bool) Option {
	return func(o *Options) {
		o.dumpReq = dump
	}
}

func WithDumpResp(dump bool) Option {
	return func(o *Options) {
		o.dumpResp = dump
	}
}

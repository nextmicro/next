package logging

import (
	"time"
)

// Option is  option.
type Option func(options *options)

type options struct {
	disabled      bool
	request       bool
	response      bool
	namespace     string
	SlowThreshold time.Duration
}

func WithDisabled(v bool) Option {
	return func(o *options) {
		o.disabled = v
	}
}

func WithRequest(v bool) Option {
	return func(o *options) {
		o.request = v
	}
}

func WithResponse(v bool) Option {
	return func(o *options) {
		o.response = v
	}
}

func WithNamespace(namespace string) Option {
	return func(o *options) {
		o.namespace = namespace
	}
}

func WithSlowThreshold(threshold time.Duration) Option {
	return func(o *options) {
		o.SlowThreshold = threshold
	}
}

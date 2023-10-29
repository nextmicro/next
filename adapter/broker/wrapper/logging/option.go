package logging

import (
	"time"
)

// Option is  option.
type Option func(options *options)

type options struct {
	addr          string        // address
	queue         string        // queue
	request       bool          // log request body
	response      bool          // log response body
	SlowThreshold time.Duration // slow time threshold
}

// WithAddr with address.
func WithAddr(addr string) Option {
	return func(o *options) {
		o.addr = addr
	}
}

// WithQueue with queue.
func WithQueue(queue string) Option {
	return func(o *options) {
		o.queue = queue
	}
}

// WithRequest with request.
func WithRequest(v bool) Option {
	return func(o *options) {
		o.request = v
	}
}

// WithResponse with response.
func WithResponse(v bool) Option {
	return func(o *options) {
		o.response = v
	}
}

// WithSlowThreshold with slow threshold.
func WithSlowThreshold(threshold time.Duration) Option {
	return func(o *options) {
		o.SlowThreshold = threshold
	}
}

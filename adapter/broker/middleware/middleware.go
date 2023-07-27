package middleware

import (
	"context"
)

type CallerKey struct{}

// NewCallerContext creates a new context with caller information attached.
func NewCallerContext(ctx context.Context, caller string) context.Context {
	return context.WithValue(ctx, CallerKey{}, caller)
}

// FromCallerContext returns the caller information in ctx if it exists.
func FromCallerContext(ctx context.Context) (caller string, ok bool) {
	caller, ok = ctx.Value(CallerKey{}).(string)
	return
}

// Handler defines the handler invoked by Middleware.
type Handler func(ctx context.Context, topic string, req interface{}) (interface{}, error)

// Middleware is mongo middleware.
type Middleware func(Handler) Handler

// Chain returns a Middleware that specifies the chained handler for broker.
func Chain(m ...Middleware) Middleware {
	return func(next Handler) Handler {
		for i := len(m) - 1; i >= 0; i-- {
			next = m[i](next)
		}
		return next
	}
}

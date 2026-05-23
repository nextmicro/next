package circuitbreaker

import (
	kbreaker "github.com/go-kratos/kratos/v2/middleware/circuitbreaker"
	"github.com/go-kratos/kratos/v2/middleware"
)

// Option is circuitbreaker option (kratos adapter).
type Option = kbreaker.Option

// ErrNotAllowed is request failed due to circuit breaker triggered.
var ErrNotAllowed = kbreaker.ErrNotAllowed

// WithCircuitBreaker sets the circuit breaker factory.
var WithCircuitBreaker = kbreaker.WithCircuitBreaker

// Client returns circuitbreaker client middleware via kratos adapter.
func Client(opts ...Option) middleware.Middleware {
	return kbreaker.Client(opts...)
}

package bbr

import (
	aegisBbr "github.com/go-kratos/aegis/ratelimit/bbr"
	kratelimit "github.com/go-kratos/kratos/v2/middleware/ratelimit"
)

// WithBBRLimiter configures the underlying aegis BBR limiter.
func WithBBRLimiter(opts ...aegisBbr.Option) Option {
	return kratelimit.WithLimiter(aegisBbr.NewLimiter(opts...))
}

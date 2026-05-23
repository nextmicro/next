package bbr

import (
	kratelimit "github.com/go-kratos/kratos/v2/middleware/ratelimit"
	"github.com/go-kratos/kratos/v2/middleware"
)

// Option is ratelimit option (kratos adapter).
type Option = kratelimit.Option

// ErrLimitExceed is service unavailable due to rate limit exceeded.
var ErrLimitExceed = kratelimit.ErrLimitExceed

// Server returns BBR ratelimit server middleware via kratos ratelimit adapter.
func Server(opts ...Option) middleware.Middleware {
	return kratelimit.Server(opts...)
}

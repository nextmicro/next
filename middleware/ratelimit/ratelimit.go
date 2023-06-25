package ratelimit

import (
	"context"

	"github.com/go-kratos/aegis/ratelimit"
	"github.com/go-kratos/aegis/ratelimit/bbr"
	"github.com/go-kratos/kratos/v2/errors"
	middlewa "github.com/go-kratos/kratos/v2/middleware"
	config "github.com/nextmicro/next/api/config/v1"
	"github.com/nextmicro/next/middleware"
)

func init() {
	middleware.Register("bbr", Server)
}

// ErrLimitExceed is service unavailable due to rate limit exceeded.
var ErrLimitExceed = errors.New(429, "RATELIMIT", "service unavailable due to rate limit exceeded")

func Server(c *config.Middleware) (middlewa.Middleware, error) {
	limiter := bbr.NewLimiter() //use default settings
	return func(handler middlewa.Handler) middlewa.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			done, e := limiter.Allow()
			if e != nil {
				// rejected
				return nil, ErrLimitExceed
			}
			// allowed
			reply, err = handler(ctx, req)
			done(ratelimit.DoneInfo{Err: err})
			return reply, err
		}
	}, nil
}

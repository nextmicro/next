package bbr

import (
	"context"

	"github.com/go-kratos/aegis/ratelimit"
	"github.com/go-kratos/aegis/ratelimit/bbr"
	"github.com/go-kratos/kratos/v2/errors"
	chain "github.com/go-kratos/kratos/v2/middleware"
	config "github.com/nextmicro/next/api/config/v1"
	v1 "github.com/nextmicro/next/api/middleware/bbr/v1"
	"github.com/nextmicro/next/middleware"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func init() {
	middleware.Register("bbr.server", injection)
}

// ErrLimitExceed is service unavailable due to rate limit exceeded.
var ErrLimitExceed = errors.New(429, "RATELIMIT", "service unavailable due to rate limit exceeded")

func injection(c *config.Middleware) (chain.Middleware, error) {
	cfg := &v1.BBR{}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, cfg, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}

	opts := make([]bbr.Option, 0)
	if cfg.GetWindow().AsDuration() >= 0 {
		opts = append(opts, bbr.WithWindow(cfg.Window.AsDuration()))
	}
	if cfg.GetBucket() != 0 {
		opts = append(opts, bbr.WithBucket(int(cfg.Bucket)))
	}
	if cfg.CpuThreshold != 0 {
		opts = append(opts, bbr.WithCPUThreshold(cfg.CpuThreshold))
	}
	if cfg.CpuQuota != 0 {
		opts = append(opts, bbr.WithCPUQuota(cfg.CpuQuota))
	}
	return Server(opts...), nil
}

func Server(opts ...bbr.Option) chain.Middleware {
	limiter := bbr.NewLimiter(opts...) // use default settings
	return func(handler chain.Handler) chain.Handler {
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
	}
}

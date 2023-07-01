package bbr

import (
	"context"

	"github.com/go-kratos/aegis/ratelimit"
	"github.com/go-kratos/aegis/ratelimit/bbr"
	"github.com/go-kratos/kratos/v2/errors"
	middlewa "github.com/go-kratos/kratos/v2/middleware"
	config "github.com/nextmicro/next/api/config/v1"
	v1 "github.com/nextmicro/next/api/middleware/bbr"
	"github.com/nextmicro/next/middleware"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func init() {
	middleware.Register("bbr.server", Server)
}

// ErrLimitExceed is service unavailable due to rate limit exceeded.
var ErrLimitExceed = errors.New(429, "RATELIMIT", "service unavailable due to rate limit exceeded")

func Server(c *config.Middleware) (middlewa.Middleware, error) {
	cfg := &v1.BBR{}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, cfg, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}

	var bbrOpts []bbr.Option
	if cfg.GetWindow().AsDuration() >= 0 {
		bbrOpts = append(bbrOpts, bbr.WithWindow(cfg.Window.AsDuration()))
	}
	if cfg.GetBucket() != 0 {
		bbrOpts = append(bbrOpts, bbr.WithBucket(int(cfg.Bucket)))
	}
	if cfg.CpuThreshold != 0 {
		bbrOpts = append(bbrOpts, bbr.WithCPUThreshold(cfg.CpuThreshold))
	}
	if cfg.CpuQuota != 0 {
		bbrOpts = append(bbrOpts, bbr.WithCPUQuota(cfg.CpuQuota))
	}

	limiter := bbr.NewLimiter(bbrOpts...) // use default settings
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

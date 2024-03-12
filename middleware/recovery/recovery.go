package recovery

import (
	"context"
	"runtime"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/nextmicro/logger"
	config "github.com/nextmicro/next/api/config/v1"
	v1 "github.com/nextmicro/next/api/middleware/recovery"
	chain "github.com/nextmicro/next/middleware"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func init() {
	chain.Register("client.recovery", injection)
	chain.Register("server.recovery", injection)
}

// ErrPanicRecover is panic recover error.
var ErrPanicRecover = errors.InternalServer("PANIC_RECOVER", "Panic recover error")

func injection(c *config.Middleware) (middleware.Middleware, error) {
	cfg := &v1.Recovery{
		StackSize:         5 << 10,
		DisableStackAll:   false,
		DisablePrintStack: false,
	}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, cfg, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}

	opts := make([]Option, 0)
	if cfg.StackSize > 0 {
		opts = append(opts, WithStackSize(int(cfg.StackSize)))
	}
	if cfg.DisableStackAll {
		opts = append(opts, WithDisableStackAll(cfg.DisableStackAll))
	}
	if cfg.DisablePrintStack {
		opts = append(opts, WithDisablePrintStack(cfg.DisablePrintStack))
	}

	return Recovery(opts...), nil
}

// Recovery is a server middleware that recovers from any panics.
func Recovery(opts ...Option) middleware.Middleware {
	cfg := options{
		logger:            logger.DefaultLogger,
		stackSize:         5 << 10,
		disableStackAll:   false,
		disablePrintStack: false,
		handler: func(ctx context.Context, err interface{}) error {
			return ErrPanicRecover
		},
	}
	for _, o := range opts {
		o(&cfg)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (resp interface{}, err error) {

			defer func() {
				if rerr := recover(); rerr != nil {
					buf := make([]byte, cfg.stackSize)
					length := runtime.Stack(buf, !cfg.disableStackAll)
					buf = buf[:length]

					err = cfg.handler(ctx, rerr)
					logger.WithContext(ctx).Errorf("[PANIC RECOVER] error: %v,  stack: %s", rerr, buf)
				}
			}()
			return handler(ctx, req)
		}
	}
}

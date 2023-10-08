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
	chain.Register("recovery.*", Recovery)
}

// ErrUnknownRequest is unknown request error.
var ErrUnknownRequest = errors.InternalServer("UNKNOWN", "unknown request error")

// Recovery is a server middleware that recovers from any panics.
func Recovery(c *config.Middleware) (middleware.Middleware, error) {
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

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (resp interface{}, err error) {

			defer func() {
				if rerr := recover(); rerr != nil {
					buf := make([]byte, cfg.StackSize)
					length := runtime.Stack(buf, !cfg.DisableStackAll)
					buf = buf[:length]

					err = ErrUnknownRequest
					logger.WithContext(ctx).Errorf("[PANIC RECOVER] error: %v,  stack: %s", rerr, buf)
				}
			}()
			return handler(ctx, req)
		}
	}, nil
}

package recovery

import (
	"context"
	"fmt"
	"runtime"

	"github.com/nextmicro/logger"
	configv1 "github.com/nextmicro/next/api/config/v1"
	v1 "github.com/nextmicro/next/api/middleware/recovery"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func optionsFromConfig(c *configv1.Middleware) ([]Option, error) {
	cfg := &v1.Recovery{
		StackSize:         5 << 10,
		DisableStackAll:   false,
		DisablePrintStack: false,
	}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, cfg, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, fmt.Errorf("recovery: unmarshal options: %w", err)
		}
	}

	stackSize := int(cfg.StackSize)
	if stackSize <= 0 {
		stackSize = 5 << 10
	}
	disableStackAll := cfg.DisableStackAll
	disablePrintStack := cfg.DisablePrintStack

	return []Option{
		WithHandler(func(ctx context.Context, _ any, rerr any) error {
			if !disablePrintStack {
				buf := make([]byte, stackSize)
				length := runtime.Stack(buf, !disableStackAll)
				logger.WithContext(ctx).Errorf("[PANIC RECOVER] error: %v, stack: %s", rerr, buf[:length])
			}
			return ErrPanicRecover
		}),
	}, nil
}

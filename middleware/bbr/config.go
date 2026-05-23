package bbr

import (
	"fmt"

	aegisBbr "github.com/go-kratos/aegis/ratelimit/bbr"
	configv1 "github.com/nextmicro/next/api/config/v1"
	v1 "github.com/nextmicro/next/api/middleware/bbr/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func optionsFromConfig(c *configv1.Middleware) ([]Option, error) {
	cfg := &v1.BBR{}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, cfg, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, fmt.Errorf("bbr: unmarshal options: %w", err)
		}
	}

	bbrOpts := make([]aegisBbr.Option, 0, 4)
	if cfg.GetWindow().AsDuration() >= 0 {
		bbrOpts = append(bbrOpts, aegisBbr.WithWindow(cfg.Window.AsDuration()))
	}
	if cfg.GetBucket() != 0 {
		bbrOpts = append(bbrOpts, aegisBbr.WithBucket(int(cfg.Bucket)))
	}
	if cfg.CpuThreshold != 0 {
		bbrOpts = append(bbrOpts, aegisBbr.WithCPUThreshold(cfg.CpuThreshold))
	}
	if cfg.CpuQuota != 0 {
		bbrOpts = append(bbrOpts, aegisBbr.WithCPUQuota(cfg.CpuQuota))
	}

	return []Option{WithBBRLimiter(bbrOpts...)}, nil
}

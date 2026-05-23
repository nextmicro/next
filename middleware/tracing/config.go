package tracing

import (
	"fmt"

	configv1 "github.com/nextmicro/next/api/config/v1"
	v1 "github.com/nextmicro/next/api/middleware/tracing/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func optionsFromConfig(c *configv1.Middleware) ([]Option, error) {
	cfg := &v1.Tracing{}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, cfg, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, fmt.Errorf("tracing: unmarshal options: %w", err)
		}
	}
	opts := make([]Option, 0, 1)
	if cfg.Disabled {
		opts = append(opts, WithDisabled(true))
	}
	return opts, nil
}

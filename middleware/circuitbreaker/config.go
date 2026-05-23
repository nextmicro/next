package circuitbreaker

import (
	"fmt"

	configv1 "github.com/nextmicro/next/api/config/v1"
	v1 "github.com/nextmicro/next/api/middleware/circuitbreaker/v1"
	"github.com/go-kratos/aegis/circuitbreaker"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func optionsFromConfig(c *configv1.Middleware) ([]Option, error) {
	cfg := &v1.CircuitBreaker{}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, cfg, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, fmt.Errorf("circuitbreaker: unmarshal options: %w", err)
		}
	}

	triggerCfg := cfg
	return []Option{
		WithCircuitBreaker(func() circuitbreaker.CircuitBreaker {
			return makeBreakerTrigger(triggerCfg)
		}),
	}, nil
}

package metadata

import (
	"fmt"

	configv1 "github.com/nextmicro/next/api/config/v1"
	v1 "github.com/nextmicro/next/api/middleware/metadata/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func optionsFromConfigServer(c *configv1.Middleware) ([]Option, error) {
	cfg := &v1.Metadata{Prefix: []string{"x-md-"}}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, cfg, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, fmt.Errorf("metadata: unmarshal server options: %w", err)
		}
	}
	opts := make([]Option, 0, 1)
	if len(cfg.Prefix) > 0 {
		opts = append(opts, WithPropagatedPrefix(cfg.Prefix...))
	}
	return opts, nil
}

func optionsFromConfigClient(c *configv1.Middleware) ([]Option, error) {
	cfg := &v1.Metadata{Prefix: []string{"x-md-global-"}}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, cfg, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, fmt.Errorf("metadata: unmarshal client options: %w", err)
		}
	}
	opts := make([]Option, 0, 1)
	if len(cfg.Prefix) > 0 {
		opts = append(opts, WithPropagatedPrefix(cfg.Prefix...))
	}
	return opts, nil
}

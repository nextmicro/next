package logger

import (
	"context"

	log "github.com/nextmicro/logger"
	config "github.com/nextmicro/next/api/config/v1"
	"github.com/nextmicro/next/runtime/loader"
)

type loggerKey struct{}

// WithConfig sets the logger config
func WithConfig(cfg *config.Logger) loader.Option {
	return func(o *loader.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, loggerKey{}, cfg)
	}
}

func options(c *config.Logger) []log.Option {
	var opts []log.Option
	if c.Path != "" {
		opts = append(opts, log.WithPath(c.Path))
	}
	if c.Level != "" {
		opts = append(opts, log.WithLevel(log.ParseLevel(c.Level)))
	}
	if c.File {
		opts = append(opts, log.WithMode(log.FileMode))
	}
	md := make(map[string]interface{})
	for k, v := range c.Metadata {
		md[k] = v
	}

	opts = append(opts,
		log.Fields(md),
	)
	return opts
}

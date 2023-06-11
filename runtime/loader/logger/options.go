package logger

import (
	"context"

	log "github.com/go-volo/logger"
	config "github.com/nextmicro/next/api/config"
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
		opts = append(opts, log.WithBasePath(c.Path))
	}
	if c.Level != "" {
		opts = append(opts, log.WithLevel(log.ParseLevel(c.Level)))
	}

	md := make(map[string]interface{})
	for k, v := range c.Metadata {
		md[k] = v
	}
	opts = append(opts,
		log.WithConsole(c.Console),
		log.WithDisableDisk(!c.File),
		log.WithFields(md),
	)
	return opts
}

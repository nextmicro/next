package runtime

import (
	"github.com/nextmicro/next/runtime/loader"
	"github.com/nextmicro/next/runtime/loader/broker"
	"github.com/nextmicro/next/runtime/loader/logger"
	"github.com/nextmicro/next/runtime/loader/registry"
	"github.com/nextmicro/next/runtime/loader/tracing"
)

type Option func(o *Options)

// Options configure runtime
type Options struct {
	loader []loader.Loader
}

// applyOptions configure runtime
func applyOptions(opts ...Option) Options {
	options := Options{
		loader: []loader.Loader{
			logger.New(),   // logger loader
			tracing.New(),  // tracing loader
			registry.New(), // registry loader
			broker.New(),   // broker loader
		},
	}

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	return options
}

// Loader with service loader.
func Loader(loader ...loader.Loader) Option {
	return func(o *Options) { o.loader = append(o.loader, loader...) }
}

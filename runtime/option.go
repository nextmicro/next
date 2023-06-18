package runtime

import (
	"github.com/nextmicro/next/runtime/loader"
	"github.com/nextmicro/next/runtime/loader/logger"
	"github.com/nextmicro/next/runtime/loader/tracing"
)

type Option func(o *Options)

// Options configure runtime
type Options struct {
	id       string
	name     string
	version  string
	metadata map[string]string
	loader   []loader.Loader
}

// defaultOptions configure runtime
func defaultOptions(opts ...Option) Options {
	options := Options{
		loader: []loader.Loader{
			logger.New(),  // logger loader
			tracing.New(), // tracing loader
		},
		metadata: make(map[string]string),
	}

	// apply requested options
	for _, o := range opts {
		o(&options)
	}

	return options
}

// ID with service id.
func ID(id string) Option {
	return func(o *Options) { o.id = id }
}

// Name with service name.
func Name(name string) Option {
	return func(o *Options) { o.name = name }
}

// Version with service version.
func Version(version string) Option {
	return func(o *Options) { o.version = version }
}

// Metadata with service metadata.
func Metadata(md map[string]string) Option {
	return func(o *Options) { o.metadata = md }
}

// Loader with service loader.
func Loader(loader ...loader.Loader) Option {
	return func(o *Options) { o.loader = append(o.loader, loader...) }
}

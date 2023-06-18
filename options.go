package next

import (
	"context"
	"net/url"
	"os"
	"time"

	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-volo/logger"
	"github.com/nextmicro/next/runtime/loader"
)

// Option is an application option.
type Option func(o *Options)

// Options is an application options.
type Options struct {
	id        string
	name      string
	version   string
	metadata  map[string]string
	endpoints []*url.URL

	ctx  context.Context
	sigs []os.Signal

	logger           logger.Logger
	registrar        registry.Registrar
	registrarTimeout time.Duration
	stopTimeout      time.Duration
	loader           []loader.Loader
	servers          []transport.Server

	// Before and After funcs
	beforeStart []func(context.Context) error
	beforeStop  []func(context.Context) error
	afterStart  []func(context.Context) error
	afterStop   []func(context.Context) error
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

// Endpoint with service endpoint.
func Endpoint(endpoints ...*url.URL) Option {
	return func(o *Options) { o.endpoints = endpoints }
}

// Context with service context.
func Context(ctx context.Context) Option {
	return func(o *Options) { o.ctx = ctx }
}

// Logger with service logger.
func Logger(logger logger.Logger) Option {
	return func(o *Options) { o.logger = logger }
}

// Loader with service loader.
func Loader(loader ...loader.Loader) Option {
	return func(o *Options) { o.loader = append(o.loader, loader...) }
}

// Server with transport servers.
func Server(srv ...transport.Server) Option {
	return func(o *Options) { o.servers = srv }
}

// Signal with exit signals.
func Signal(sigs ...os.Signal) Option {
	return func(o *Options) { o.sigs = sigs }
}

// Registrar with service registry.
func Registrar(r registry.Registrar) Option {
	return func(o *Options) { o.registrar = r }
}

// RegistrarTimeout with registrar timeout.
func RegistrarTimeout(t time.Duration) Option {
	return func(o *Options) { o.registrarTimeout = t }
}

// StopTimeout with app stop timeout.
func StopTimeout(t time.Duration) Option {
	return func(o *Options) { o.stopTimeout = t }
}

// Before and Afters

// BeforeStart run funcs before app starts
func BeforeStart(fn func(context.Context) error) Option {
	return func(o *Options) {
		o.beforeStart = append(o.beforeStart, fn)
	}
}

// BeforeStop run funcs before app stops
func BeforeStop(fn func(context.Context) error) Option {
	return func(o *Options) {
		o.beforeStop = append(o.beforeStop, fn)
	}
}

// AfterStart run funcs after app starts
func AfterStart(fn func(context.Context) error) Option {
	return func(o *Options) {
		o.afterStart = append(o.afterStart, fn)
	}
}

// AfterStop run funcs after app stops
func AfterStop(fn func(context.Context) error) Option {
	return func(o *Options) {
		o.afterStop = append(o.afterStop, fn)
	}
}

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
	ID        string
	Name      string
	Version   string
	Metadata  map[string]string
	Endpoints []*url.URL

	Ctx  context.Context
	Sigs []os.Signal

	Logger           logger.Logger
	Registrar        registry.Registrar
	RegistrarTimeout time.Duration
	StopTimeout      time.Duration
	Loader           []loader.Loader
	Servers          []transport.Server

	// Before and After funcs
	BeforeStart []func(context.Context) error
	BeforeStop  []func(context.Context) error
	AfterStart  []func(context.Context) error
	AfterStop   []func(context.Context) error
}

// ID with service ID.
func ID(id string) Option {
	return func(o *Options) { o.ID = id }
}

// Name with service Name.
func Name(name string) Option {
	return func(o *Options) { o.Name = name }
}

// Version with service Version.
func Version(version string) Option {
	return func(o *Options) { o.Version = version }
}

// Metadata with service Metadata.
func Metadata(md map[string]string) Option {
	return func(o *Options) { o.Metadata = md }
}

// Endpoint with service endpoint.
func Endpoint(endpoints ...*url.URL) Option {
	return func(o *Options) { o.Endpoints = endpoints }
}

// Context with service context.
func Context(ctx context.Context) Option {
	return func(o *Options) { o.Ctx = ctx }
}

// Logger with service Logger.
func Logger(logger logger.Logger) Option {
	return func(o *Options) { o.Logger = logger }
}

// Loader with service Loader.
func Loader(loader ...loader.Loader) Option {
	return func(o *Options) { o.Loader = append(o.Loader, loader...) }
}

// Server with transport Servers.
func Server(srv ...transport.Server) Option {
	return func(o *Options) { o.Servers = srv }
}

// Signal with exit signals.
func Signal(sigs ...os.Signal) Option {
	return func(o *Options) { o.Sigs = sigs }
}

// Registrar with service registry.
func Registrar(r registry.Registrar) Option {
	return func(o *Options) { o.Registrar = r }
}

// RegistrarTimeout with Registrar timeout.
func RegistrarTimeout(t time.Duration) Option {
	return func(o *Options) { o.RegistrarTimeout = t }
}

// StopTimeout with app stop timeout.
func StopTimeout(t time.Duration) Option {
	return func(o *Options) { o.StopTimeout = t }
}

// Before and Afters

// BeforeStart run funcs before app starts
func BeforeStart(fn func(context.Context) error) Option {
	return func(o *Options) {
		o.BeforeStart = append(o.BeforeStart, fn)
	}
}

// BeforeStop run funcs before app stops
func BeforeStop(fn func(context.Context) error) Option {
	return func(o *Options) {
		o.BeforeStop = append(o.BeforeStop, fn)
	}
}

// AfterStart run funcs after app starts
func AfterStart(fn func(context.Context) error) Option {
	return func(o *Options) {
		o.AfterStart = append(o.AfterStart, fn)
	}
}

// AfterStop run funcs after app stops
func AfterStop(fn func(context.Context) error) Option {
	return func(o *Options) {
		o.AfterStop = append(o.AfterStop, fn)
	}
}

package transport

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/middleware"
	v1 "github.com/nextmicro/next/api/config"
	conf "github.com/nextmicro/next/config"
)

type ServerOption interface {
	Apply(o *Options, cfg *v1.Next) error
}

type OptionFunc func(*Options, *v1.Next) error

func (fn OptionFunc) Apply(o *Options, cfg *v1.Next) error {
	return fn(o, cfg)
}

type Options struct {
	Address    string                  // server address
	Timeout    time.Duration           // server timeout
	Context    context.Context         // server context
	Middleware []middleware.Middleware // server middleware
}

// NewDefaultOptions returns a new Options with default values.
func NewDefaultOptions(cfg *v1.Next, opts ...ServerOption) (*Options, error) {
	op := &Options{
		Context: context.Background(),
		Timeout: 1 * time.Second,
	}

	// use custom config
	for _, o := range opts {
		if err := o.Apply(op, conf.AppConfig()); err != nil {
			return nil, err
		}
	}

	return op, nil
}

// Address with server address.
func Address(address string) ServerOption {
	return OptionFunc(func(o *Options, cfg *v1.Next) error {
		o.Address = address
		return nil
	})
}

// Timeout with server timeout.
func Timeout(timeout time.Duration) ServerOption {
	return OptionFunc(func(o *Options, cfg *v1.Next) error {
		o.Timeout = timeout
		return nil
	})
}

// Middleware with server middleware.
func Middleware(m ...middleware.Middleware) ServerOption {
	return OptionFunc(func(o *Options, cfg *v1.Next) error {
		o.Middleware = append(o.Middleware, m...)
		return nil
	})
}

package middleware

import (
	"errors"
	"strings"

	"github.com/go-kratos/kratos/v2/middleware"
	log "github.com/go-volo/logger"
	configv1 "github.com/nextmicro/next/api/config/v1"
)

var globalRegistry = NewRegistry()

// ErrNotFound is middleware not found.
var ErrNotFound = errors.New("Middleware has not been registered")

// Factory is a middleware factory.
type Factory func(*configv1.Middleware) (middleware.Middleware, error)

// Registry is the interface for callers to get registered middleware.
type Registry interface {
	Register(name string, factory Factory)
	Create(cfg *configv1.Middleware) (middleware.Middleware, error)
}

type middlewareRegistry struct {
	middleware map[string]Factory
}

// NewRegistry returns a new middleware registry.
func NewRegistry() Registry {
	return &middlewareRegistry{
		middleware: map[string]Factory{},
	}
}

// Register registers one middleware.
func (p *middlewareRegistry) Register(name string, factory Factory) {
	p.middleware[createFullName(name)] = factory
}

// Create instantiates a middleware based on `cfg`.
func (p *middlewareRegistry) Create(cfg *configv1.Middleware) (middleware.Middleware, error) {
	if method, ok := p.getMiddleware(createFullName(cfg.Name)); ok {
		return method(cfg)
	}
	return nil, ErrNotFound
}

func (p *middlewareRegistry) getMiddleware(name string) (Factory, bool) {
	nameLower := strings.ToLower(name)
	middlewareFn, ok := p.middleware[nameLower]
	if ok {
		return middlewareFn, true
	}
	return nil, false
}

func createFullName(name string) string {
	return strings.ToLower("next.middleware." + name)
}

// Register registers one middleware.
func Register(name string, factory Factory) {
	globalRegistry.Register(name, factory)
}

// Create instantiates a middleware based on `cfg`.
func Create(cfg *configv1.Middleware) (middleware.Middleware, error) {
	return globalRegistry.Create(cfg)
}

func BuildMiddleware(kind string, _ms []*configv1.Middleware) (ms []middleware.Middleware, err error) {
	for i := len(_ms) - 1; i >= 0; i-- {
		_ms[i].Name = kind + "." + _ms[i].Name
		m, err := Create(_ms[i])
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				log.Warnf("Skip does not exist middleware: %s", _ms[i].Name)
				continue
			}
			return nil, err
		}

		log.Infof("register middleware: [%s] success", _ms[i].Name)

		ms = append(ms, m)
	}
	return ms, nil
}

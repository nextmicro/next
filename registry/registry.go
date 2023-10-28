package registry

import (
	"errors"

	"github.com/go-kratos/kratos/v2/registry"
)

var (
	DefaultRegistry = NewRegistry()

	// ErrWatcherStopped error when watcher is stopped.
	ErrWatcherStopped = errors.New("watcher stopped")
)

// Registry The registry provides an interface for service discovery
// and an abstraction over varying implementations
// {consul, etcd, zookeeper, ...}.
type Registry interface {
	registry.Registrar
	registry.Discovery
}

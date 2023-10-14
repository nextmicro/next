package registry

import (
	"context"
	"sync"

	"github.com/go-kratos/kratos/v2/registry"
)

type memory struct {
	sync.RWMutex
	records map[string]*registry.ServiceInstance
}

func NewMemory() Registry {
	return &memory{
		records: make(map[string]*registry.ServiceInstance),
	}
}

func (r *memory) Register(ctx context.Context, service *registry.ServiceInstance) error {
	r.Lock()
	defer r.Unlock()

	r.records[service.Name] = service
	return nil
}

func (r *memory) Deregister(ctx context.Context, service *registry.ServiceInstance) error {
	r.Lock()
	defer r.Unlock()

	delete(r.records, service.Name)
	return nil
}

func (r *memory) GetService(ctx context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	r.Lock()
	defer r.Unlock()

	var instances []*registry.ServiceInstance
	instance, ok := r.records[serviceName]
	if ok {
		instances = append(instances, instance)
	}
	return instances, nil
}

func (r *memory) Watch(ctx context.Context, serviceName string) (registry.Watcher, error) {
	return &memoryWatcher{}, nil
}

type memoryWatcher struct{}

func (m *memoryWatcher) Next() ([]*registry.ServiceInstance, error) {
	return nil, nil
}

func (m *memoryWatcher) Stop() error {
	return nil
}

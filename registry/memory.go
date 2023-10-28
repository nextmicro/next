package registry

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/registry"
	"github.com/google/uuid"
	"github.com/nacos-group/nacos-sdk-go/v2/common/logger"
)

var (
	sendEventTime = 10 * time.Millisecond
	ttlPruneTime  = time.Second
)

type record struct {
	*registry.ServiceInstance
	TTL      time.Duration
	LastSeen time.Time
}

type memRegistry struct {
	sync.RWMutex
	records  map[string]*record
	watchers map[string]*memoryWatcher
}

// Result is returned by a call to Next on
// the watcher. Actions can be create, update, delete.
type Result struct {
	Action  string
	Service *registry.ServiceInstance
}

func NewMemory() Registry {
	return &memRegistry{
		records: make(map[string]*record),
	}
}

func serviceToRecord(s *registry.ServiceInstance, ttl time.Duration) *record {
	metadata := make(map[string]string, len(s.Metadata))
	for k, v := range s.Metadata {
		metadata[k] = v
	}

	return &record{
		ServiceInstance: s,
		TTL:             ttl,
		LastSeen:        time.Now(),
	}
}

func (r *memRegistry) Register(ctx context.Context, service *registry.ServiceInstance) error {
	r.Lock()
	defer r.Unlock()

	record := serviceToRecord(service, 0)
	r.records[service.Name] = record
	go r.sendEvent(&Result{Action: "update", Service: service})
	logger.Infof("Registry added new service: %s, version: %s", service.Name, service.Version)

	return nil
}

func (r *memRegistry) sendEvent(result *Result) {
	r.RLock()
	watchers := make([]*memoryWatcher, 0, len(r.watchers))
	for _, w := range r.watchers {
		watchers = append(watchers, w)
	}
	r.RUnlock()

	for _, w := range watchers {
		select {
		case <-w.exit:
			r.Lock()
			delete(r.watchers, w.id)
			r.Unlock()
		default:
			select {
			case w.res <- result:
			case <-time.After(sendEventTime):
			}
		}
	}
}

func (r *memRegistry) Deregister(ctx context.Context, service *registry.ServiceInstance) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.records[service.Name]; ok {
		logger.Infof("Registry removed node from service: %s, version: %s", service.Name, service.Version)
		delete(r.records, service.Name)
	}

	go r.sendEvent(&Result{Action: "delete", Service: service})

	return nil
}

func (r *memRegistry) GetService(ctx context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	r.Lock()
	defer r.Unlock()

	record, ok := r.records[serviceName]
	if !ok {
		return nil, fmt.Errorf("service %s not found in registry", serviceName)
	}

	var instances []*registry.ServiceInstance
	instances = []*registry.ServiceInstance{record.ServiceInstance}

	return instances, nil
}

func (r *memRegistry) Watch(ctx context.Context, serviceName string) (registry.Watcher, error) {
	w := &memoryWatcher{
		exit:    make(chan bool),
		res:     make(chan *Result),
		id:      uuid.New().String(),
		service: serviceName,
	}

	r.Lock()
	r.watchers[w.id] = w
	r.Unlock()

	return w, nil
}

type memoryWatcher struct {
	id      string
	service string
	res     chan *Result
	exit    chan bool
}

func (m *memoryWatcher) Next() ([]*registry.ServiceInstance, error) {
	for {
		select {
		case r := <-m.res:
			if len(m.service) > 0 && m.service != r.Service.Name {
				continue
			}
			return []*registry.ServiceInstance{r.Service}, nil
		case <-m.exit:
			return nil, errors.New("watcher stopped")
		}
	}
}

func (m *memoryWatcher) Stop() error {
	select {
	case <-m.exit:
		return nil
	default:
		close(m.exit)
	}

	return nil
}

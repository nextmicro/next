package registry

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/registry"
	log "github.com/go-volo/logger"
	"github.com/nextmicro/next/pkg/mdns"
)

var (
	// use a .next domain rather than .local.
	mdnsDomain = "next"
)

type mdnsTxt struct {
	Service  string
	Version  string
	Metadata map[string]string
}

type mdnsEntry struct {
	id   string
	node *mdns.Server
}

type mdnsRegistry struct {
	// the mdns domain
	domain string

	services map[string][]*mdnsEntry

	mtx sync.RWMutex

	// listener
	listener chan *mdns.ServiceEntry
}

type watcher struct {
	ctx        context.Context
	cancel     context.CancelFunc
	domain     string
	serverName string
	watchChan  chan *mdns.ServiceEntry
	exit       chan struct{}
}

func newRegistry() Registry {
	// set the domain
	domain := mdnsDomain

	return &mdnsRegistry{
		domain:   domain,
		services: make(map[string][]*mdnsEntry),
	}
}

func encode(txt *mdnsTxt) ([]string, error) {
	b, err := json.Marshal(txt)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	defer buf.Reset()

	w := zlib.NewWriter(&buf)
	if _, err := w.Write(b); err != nil {
		return nil, err
	}
	w.Close()

	encoded := hex.EncodeToString(buf.Bytes())

	// individual txt limit
	if len(encoded) <= 255 {
		return []string{encoded}, nil
	}

	// split encoded string
	var record []string

	for len(encoded) > 255 {
		record = append(record, encoded[:255])
		encoded = encoded[255:]
	}

	record = append(record, encoded)

	return record, nil
}

func decode(record []string) (*mdnsTxt, error) {
	encoded := strings.Join(record, "")

	hr, err := hex.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	br := bytes.NewReader(hr)
	zr, err := zlib.NewReader(br)
	if err != nil {
		return nil, err
	}

	rbuf, err := io.ReadAll(zr)
	if err != nil {
		return nil, err
	}

	var txt *mdnsTxt

	if err := json.Unmarshal(rbuf, &txt); err != nil {
		return nil, err
	}

	return txt, nil
}

func (m *mdnsRegistry) Register(ctx context.Context, service *registry.ServiceInstance) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	entries, ok := m.services[service.Name]
	// first entry, create wildcard used for list queries
	if !ok {
		s, err := mdns.NewMDNSService(
			service.Name,
			"_services",
			m.domain+".",
			"",
			9999,
			[]net.IP{net.ParseIP("0.0.0.0")},
			nil,
		)
		if err != nil {
			return err
		}

		srv, err := mdns.NewServer(&mdns.Config{Zone: &mdns.DNSSDService{MDNSService: s}})
		if err != nil {
			return err
		}

		// append the wildcard entry
		entries = append(entries, &mdnsEntry{id: "*", node: srv})
	}

	var gerr error

	for _, endpoint := range service.Endpoints {
		var seen bool
		var e *mdnsEntry

		for _, entry := range entries {
			if service.ID == entry.id {
				seen = true
				e = entry
				break
			}
		}

		// already registered, continue
		if seen {
			continue
			// doesn't exist
		} else {
			e = &mdnsEntry{}
		}

		// get url
		u, err := url.Parse(endpoint)
		if err != nil {
			return err
		}

		// get host and port
		host, port, err := net.SplitHostPort(u.Host)
		if err != nil {
			return err
		}

		// port to int
		portNum, err := strconv.Atoi(port)
		if err != nil {
			return err
		}

		var rmd map[string]string
		if service.Metadata == nil {
			rmd = map[string]string{
				"kind":    u.Scheme,
				"version": service.Version,
			}
		} else {
			rmd = make(map[string]string, len(service.Metadata)+2)
			for k, v := range service.Metadata {
				rmd[k] = v
			}
			rmd["kind"] = u.Scheme
			rmd["version"] = service.Version
		}

		txt, err := encode(&mdnsTxt{
			Service:  service.Name,
			Version:  service.Version,
			Metadata: rmd,
		})

		if err != nil {
			gerr = err
			continue
		}

		log.Infof("[mdns] registry create new service with ip: %s for: %s", net.ParseIP(host).String(), host)

		// we got here, new node
		s, err := mdns.NewMDNSService(
			service.ID,
			service.Name,
			m.domain+".",
			"",
			portNum,
			[]net.IP{net.ParseIP(host)},
			txt,
		)
		if err != nil {
			gerr = err
			continue
		}

		srv, err := mdns.NewServer(&mdns.Config{Zone: s, LocalhostChecking: true})
		if err != nil {
			gerr = err
			continue
		}

		e.id = service.ID
		e.node = srv
		entries = append(entries, e)

		// save
		m.services[service.Name] = entries
	}

	return gerr
}

func (m *mdnsRegistry) Deregister(ctx context.Context, service *registry.ServiceInstance) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	var newEntries []*mdnsEntry

	// loop existing entries, check if any match, shutdown those that do
	for _, entry := range m.services[service.Name] {
		var remove bool

		if service.ID == entry.id {
			entry.node.Shutdown()
			remove = true
			break
		}

		// keep it?
		if !remove {
			newEntries = append(newEntries, entry)
		}
	}

	// last entry is the wildcard for list queries. Remove it.
	if len(newEntries) == 1 && newEntries[0].id == "*" {
		newEntries[0].node.Shutdown()
		delete(m.services, service.Name)
	} else {
		m.services[service.Name] = newEntries
	}

	return nil
}

func (m *mdnsRegistry) GetService(ctx context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	serviceMap := make(map[string]*mdns.ServiceEntry)
	entries := make(chan *mdns.ServiceEntry, 10)
	done := make(chan bool)

	p := mdns.DefaultParams(serviceName)
	// set context with timeout
	var cancel context.CancelFunc
	p.Context, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	// set entries channel
	p.Entries = entries
	// set the domain
	p.Domain = m.domain

	go func() {
		for {
			select {
			case e := <-entries:
				// list record so skip
				if p.Service == "_services" {
					continue
				}
				if p.Domain != m.domain {
					continue
				}
				if e.TTL == 0 {
					continue
				}

				txt, err := decode(e.InfoFields)
				if err != nil {
					continue
				}

				if txt.Service != serviceName {
					continue
				}

				serviceMap[txt.Version] = e
			case <-p.Context.Done():
				close(done)
				return
			}
		}
	}()

	// execute the query
	if err := mdns.Query(p); err != nil {
		return nil, err
	}

	// wait for completion
	<-done

	instances := make([]*registry.ServiceInstance, 0, len(serviceMap))
	for _, service := range serviceMap {
		instance, err := instanceToServiceInstance(service)
		if err != nil {
			continue
		}

		instances = append(instances, instance)
	}

	return instances, nil
}

func (m *mdnsRegistry) Watch(ctx context.Context, serviceName string) (registry.Watcher, error) {
	w := &watcher{
		ctx:        ctx,
		domain:     m.domain,
		serverName: serviceName,
		watchChan:  make(chan *mdns.ServiceEntry, 32),
		exit:       make(chan struct{}),
	}
	w.ctx, w.cancel = context.WithCancel(ctx)

	go func() {
		mdns.Listen(w.watchChan, w.exit)
	}()

	return w, nil
}

func (w *watcher) Next() ([]*registry.ServiceInstance, error) {
	select {
	case <-w.ctx.Done():
		return nil, w.ctx.Err()
	case <-w.watchChan:
		instances := make([]*mdns.ServiceEntry, 0)
		entries := make(chan *mdns.ServiceEntry, 10)
		done := make(chan bool)

		p := mdns.DefaultParams(w.serverName)
		// set context with timeout
		var cancel context.CancelFunc
		p.Context, cancel = context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		// set entries channel
		p.Entries = entries
		// set the domain
		p.Domain = w.domain

		go func() {
			for {
				select {
				case e := <-entries:
					// list record so skip
					if p.Service == "_services" {
						continue
					}
					if p.Domain != w.domain {
						continue
					}
					txt, err := decode(e.InfoFields)
					if err != nil {
						continue
					}

					if txt.Service != w.serverName {
						continue
					}

					instances = append(instances, e)
				case <-p.Context.Done():
					close(done)
					return
				}
			}
		}()

		// execute the query
		if err := mdns.Query(p); err != nil {
			return nil, err
		}

		// wait for completion
		<-done

		return instancesToServiceInstances(instances), nil
	}
}

func (w *watcher) Stop() error {
	w.cancel()

	return nil
}

// NewRegistry returns a new default registry which is mdns.
func NewRegistry() Registry {
	return newRegistry()
}

func instancesToServiceInstances(instances []*mdns.ServiceEntry) []*registry.ServiceInstance {
	serviceInstances := make([]*registry.ServiceInstance, 0, len(instances))
	for _, instance := range instances {
		if instance.TTL == 0 {
			continue
		}

		serviceInstance, err := instanceToServiceInstance(instance)
		if err != nil {
			continue
		}
		serviceInstances = append(serviceInstances, serviceInstance)
	}
	return serviceInstances
}

func instanceToServiceInstance(instance *mdns.ServiceEntry) (*registry.ServiceInstance, error) {
	txt, err := decode(instance.InfoFields)
	if err != nil {
		return nil, err
	}

	metadata := txt.Metadata
	// Usually, it won't fail in kratos if register correctly
	kind := ""
	if k, ok := metadata["kind"]; ok {
		kind = k
	}

	suffix := fmt.Sprintf(".%s.%s.", txt.Service, mdnsDomain)

	var addr string
	if len(instance.AddrV4) > 0 {
		addr = net.JoinHostPort(instance.AddrV4.String(), fmt.Sprint(instance.Port))
	} else if len(instance.AddrV6) > 0 {
		addr = net.JoinHostPort(instance.AddrV6.String(), fmt.Sprint(instance.Port))
	} else {
		addr = instance.Addr.String()
	}

	return &registry.ServiceInstance{
		ID:        strings.TrimSuffix(instance.Name, suffix),
		Name:      txt.Service,
		Version:   txt.Version,
		Metadata:  metadata,
		Endpoints: []string{fmt.Sprintf("%s://%s", kind, addr)},
	}, nil
}

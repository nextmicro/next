package registry

import (
	"context"
	"net"
	"strings"

	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	etcd "github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/hashicorp/consul/api"
	"github.com/nextmicro/logger"
	config "github.com/nextmicro/next/api/config/v1"
	conf "github.com/nextmicro/next/config"
	"github.com/nextmicro/next/registry"
	"github.com/nextmicro/next/runtime/loader"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type wrapper struct {
	*loader.Base
	initialized bool
	opt         loader.Options
	cfg         *config.Registry
}

func New(opts ...loader.Option) loader.Loader {
	options := loader.NewOptions(opts...)

	return &wrapper{
		opt: *options,
	}
}

// Init options
func (loader *wrapper) Init(opts ...loader.Option) error {
	cfg := conf.ApplicationConfig()
	loader.cfg = cfg.GetRegistry()

	// check if there are any addrs
	var addrs []string

	// iterate the options addresses
	for _, address := range strings.Split(cfg.GetRegistry().GetAddrs(), ",") {
		// check we have a port
		addr, port, err := net.SplitHostPort(address)
		if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
			port = "8500"
			addr = address
			addrs = append(addrs, net.JoinHostPort(addr, port))
		} else if err == nil {
			addrs = append(addrs, net.JoinHostPort(addr, port))
		}
	}

	switch cfg.GetRegistry().GetName() {
	case "consul":
		_config := api.DefaultNonPooledConfig()
		if len(addrs) > 0 {
			_config.Address = addrs[0]
		}

		// new consul client
		client, err := api.NewClient(_config)
		if err != nil {
			return err
		}
		// test the client
		_, err = client.Agent().Host()
		if err != nil {
			return err
		}

		// new reg with consul client
		reg := consul.New(client)
		registry.DefaultRegistry = reg
	case "etcd":
		// new etcd client
		client, err := clientv3.New(clientv3.Config{
			Endpoints: addrs,
		})
		if err != nil {
			return err
		}
		// new reg with etcd client
		reg := etcd.New(client)
		registry.DefaultRegistry = reg
	default:
		if loader.cfg == nil {
			loader.cfg = &config.Registry{
				Name: "mdns",
			}
		}
		registry.DefaultRegistry = registry.NewRegistry()
	}

	loader.initialized = true
	return nil
}

// Start the broker
func (loader *wrapper) Start(ctx context.Context) (err error) {
	if !loader.initialized {
		return
	}

	logger.Infof("Registry [%s] Registering node: %s", loader.cfg.GetName(), conf.ApplicationConfig().GetName())
	return
}

// String returns the name of broker
func (loader *wrapper) String() string {
	return "registry"
}

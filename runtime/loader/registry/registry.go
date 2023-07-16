package registry

import (
	"context"
	"github.com/go-kratos/kratos/contrib/registry/consul/v2"
	etcd "github.com/go-kratos/kratos/contrib/registry/etcd/v2"
	"github.com/go-volo/logger"
	"github.com/hashicorp/consul/api"
	config "github.com/nextmicro/next/api/config/v1"
	conf "github.com/nextmicro/next/config"
	"github.com/nextmicro/next/registry"
	"github.com/nextmicro/next/runtime/loader"
	clientv3 "go.etcd.io/etcd/client/v3"
	"strings"
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
	switch cfg.GetRegistry().GetName() {
	case "consul":
		// new consul client
		client, err := api.NewClient(api.DefaultConfig())
		if err != nil {
			return err
		}
		// new reg with consul client
		reg := consul.New(client)
		registry.DefaultRegistry = reg
	case "etcd":
		// new etcd client
		client, err := clientv3.New(clientv3.Config{
			Endpoints: strings.Split(cfg.GetRegistry().GetAddrs(), ","),
		})
		if err != nil {
			panic(err)
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

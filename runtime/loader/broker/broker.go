package broker

import (
	"context"
	config "github.com/nextmicro/next/api/config/v1"
	"github.com/nextmicro/next/broker"
	conf "github.com/nextmicro/next/config"
	"github.com/nextmicro/next/internal/adapter/broker/kafka"
	"github.com/nextmicro/next/runtime/loader"
)

type wrapper struct {
	initialized bool
	opt         loader.Options
	cfg         *config.Logger
}

func New(opts ...loader.Option) loader.Loader {
	options := loader.NewOptions(opts...)

	return &wrapper{
		opt: *options,
	}
}

// Init options
func (loader *wrapper) Init(opts ...loader.Option) error {
	cfg := conf.ApplicationConfig().GetBroker()

	if cfg == nil || len(cfg.Addrs) == 0 {
		return nil
	}

	switch cfg.Name {
	case "kafka":
		kafka.New(
			broker.Addrs(cfg.Addrs...),
			kafka.Group(cfg.Group),
		)
	case "nsq":

	default:
		broker.DefaultBroker = broker.NewMemoryBroker()
		return nil
	}

	loader.initialized = true
	return nil
}

// Start the broker
func (loader *wrapper) Start(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

// Watch the broker
func (loader *wrapper) Watch() error {
	//TODO implement me
	panic("implement me")
}

// Stop the broker
func (loader *wrapper) Stop(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

// String returns the name of broker
func (loader *wrapper) String() string {
	//TODO implement me
	panic("implement me")
}

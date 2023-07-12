package broker

import (
	"context"
	"github.com/go-volo/logger"
	config "github.com/nextmicro/next/api/config/v1"
	"github.com/nextmicro/next/broker"
	conf "github.com/nextmicro/next/config"
	"github.com/nextmicro/next/internal/adapter/broker/kafka"
	"github.com/nextmicro/next/runtime/loader"
)

type wrapper struct {
	initialized bool
	opt         loader.Options
	cfg         *config.Broker
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

	var (
		queueName = cfg.GetName()
	)
	if cfg.GetBroker().GetSubscribe().GetQueue() != "" {
		queueName = cfg.GetBroker().GetSubscribe().GetQueue()
	}

	broker.DefaultBroker = broker.NewMemoryBroker()
	switch cfg.GetBroker().GetName() {
	case "kafka":
		broker.DefaultBroker = kafka.New(
			broker.Addrs(cfg.GetBroker().GetAddrs()...),
			broker.Queue(queueName),
		)
	}

	loader.cfg = cfg.GetBroker()
	loader.initialized = true
	return nil
}

// Start the broker
func (loader *wrapper) Start(ctx context.Context) (err error) {
	if !loader.initialized {
		return
	}

	if err = broker.DefaultBroker.Connect(); err != nil {
		logger.Errorf("Broker [%s] connect error: %v", broker.DefaultBroker.String(), err)
		return err
	}

	logger.Infof("Broker [%s] Connected to %s", broker.DefaultBroker.String(), broker.DefaultBroker.Address())
	return
}

// Watch the broker
func (loader *wrapper) Watch() error {
	if !loader.initialized {
		return nil
	}

	return nil
}

// Stop the broker
func (loader *wrapper) Stop(ctx context.Context) (err error) {
	if !loader.initialized {
		return nil
	}

	logger.Infof("Broker [%s] Disconnected from %s", broker.DefaultBroker.String(), broker.DefaultBroker.Address())

	// disconnect broker
	if err = broker.DefaultBroker.Disconnect(); err != nil {
		logger.Errorf("Broker [%s] disconnect error: %v", broker.DefaultBroker.String(), err)
	}

	return
}

// String returns the name of broker
func (loader *wrapper) String() string {
	return "broker"
}

package broker

import (
	"context"
	"strings"

	"github.com/nextmicro/logger"
	"github.com/nextmicro/next/adapter/broker/kafka"
	"github.com/nextmicro/next/adapter/broker/wrapper/logging"
	"github.com/nextmicro/next/adapter/broker/wrapper/metrics"
	"github.com/nextmicro/next/broker"
	conf "github.com/nextmicro/next/config"
	"github.com/nextmicro/next/runtime/loader"
	"github.com/pkg/errors"
)

type wrapper struct {
	loader.BaseLoader

	opt loader.Options
}

func New(opts ...loader.Option) loader.Loader {
	options := loader.NewOptions(opts...)

	return &wrapper{
		opt: *options,
	}
}

func (loader *wrapper) Initialized() bool {
	return loader.opt.Initialized
}

// Init options
func (loader *wrapper) Init(opts ...loader.Option) error {
	var (
		cfg       = conf.ApplicationConfig()
		brokerCfg = cfg.GetBroker()
		queueName = cfg.GetName()
	)
	if brokerCfg == nil || brokerCfg.GetDisable() {
		return nil
	}
	if len(brokerCfg.GetAddrs()) == 0 {
		return errors.New("missing broker addrs in config file")
	}

	if brokerCfg.GetSubscribe().GetQueue() != "" {
		queueName = brokerCfg.GetSubscribe().GetQueue()
	}

	brokerOpts := make([]broker.Option, 0, 2)
	brokerOpts = append(brokerOpts,
		broker.Addrs(brokerCfg.GetAddrs()...),
		broker.Queue(queueName),
		broker.Wrap(
			logging.NewWrapper(
				logging.WithAddr(strings.Join(brokerCfg.GetAddrs(), ",")),
				logging.WithQueue(queueName),
			),
			metrics.NewWrapper(
				metrics.WithAddr(strings.Join(brokerCfg.GetAddrs(), ",")),
				metrics.WithQueue(queueName),
			),
		),
	)

	switch brokerCfg.GetName() {
	case "kafka":
		broker.DefaultBroker = kafka.New(brokerOpts...)
	default:
		broker.DefaultBroker = broker.NewMemoryBroker(brokerOpts...)
	}

	loader.opt.Initialized = true
	return nil
}

// Start the broker
func (loader *wrapper) Start(ctx context.Context) (err error) {
	if err = broker.DefaultBroker.Connect(); err != nil {
		logger.Errorf("Broker [%s] connect error: %v", broker.DefaultBroker.String(), err)
		return errors.WithStack(err)
	}

	logger.Infof("Broker [%s] Connected to %s", broker.DefaultBroker.String(), broker.DefaultBroker.Address())
	return
}

// Stop the broker
func (loader *wrapper) Stop(ctx context.Context) (err error) {
	// disconnect broker
	if err = broker.DefaultBroker.Disconnect(); err != nil {
		logger.Errorf("Broker [%s] disconnect error: %v", broker.DefaultBroker.String(), err)
		return errors.WithStack(err)
	}

	logger.Infof("Broker [%s] Disconnected from %s", broker.DefaultBroker.String(), broker.DefaultBroker.Address())

	return
}

// String returns the name of broker
func (loader *wrapper) String() string {
	return "Broker"
}

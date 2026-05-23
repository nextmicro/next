// Package broker 是 next v1 兼容形态的 Broker Loader。
//
// Deprecated: 新代码请直接使用 `broker/kafka`、`broker.NewMemory` 构造 Broker
// 并通过 `lifecycle.Component` + `container.Register` 接入 App。
// 本 Loader 在 v2 过渡期仍可用，内部已切换到新 `broker/*` 与 `broker/wrappers/*` 包。
package broker

import (
	"context"
	"strings"

	"github.com/nextmicro/logger"
	"github.com/nextmicro/next/broker"
	"github.com/nextmicro/next/broker/kafka"
	"github.com/nextmicro/next/broker/wrappers/logging"
	"github.com/nextmicro/next/broker/wrappers/metrics"
	conf "github.com/nextmicro/next/config"
	"github.com/nextmicro/next/runtime/loader"
	"github.com/pkg/errors"
)

// Current 是当前 Loader 创建的 broker 实例，连接成功后非 nil。
//
// Deprecated: 仅为兼容老代码暂时保留；新代码应通过 container.Resolve[broker.Broker]
// 获取由 App 注入的实例。
var Current broker.Broker

type wrapper struct {
	loader.BaseLoader

	opt loader.Options
}

// New 创建一个 Broker Loader。
func New(opts ...loader.Option) loader.Loader {
	options := loader.NewOptions(opts...)
	return &wrapper{opt: *options}
}

// Initialized 返回 Loader 是否初始化成功。
func (l *wrapper) Initialized() bool { return l.opt.Initialized }

// Init 读取配置、构造 broker 实例（不连接）。
func (l *wrapper) Init(...loader.Option) error {
	cfg := conf.ApplicationConfig()
	brokerCfg := cfg.GetBroker()
	queueName := cfg.GetName()

	if brokerCfg == nil || brokerCfg.GetDisable() {
		return nil
	}
	if len(brokerCfg.GetAddrs()) == 0 {
		return errors.New("missing broker addrs in config file")
	}
	if brokerCfg.GetSubscribe().GetQueue() != "" {
		queueName = brokerCfg.GetSubscribe().GetQueue()
	}

	addrs := brokerCfg.GetAddrs()
	addrStr := strings.Join(addrs, ",")

	opts := []broker.Option{
		broker.Addrs(addrs...),
		broker.Wrap(
			logging.NewWrapper(
				logging.WithAddr(addrStr),
				logging.WithQueue(queueName),
			),
			metrics.NewWrapper(
				metrics.WithAddr(addrStr),
				metrics.WithQueue(queueName),
			),
		),
	}

	switch brokerCfg.GetName() {
	case "kafka":
		Current = kafka.New(opts...)
	default:
		Current = broker.NewMemory(opts...)
	}

	l.opt.Initialized = true
	return nil
}

// Start 连接 broker。
func (l *wrapper) Start(ctx context.Context) error {
	if Current == nil {
		return nil
	}
	if err := Current.Connect(ctx); err != nil {
		logger.Errorf("Broker connect error: %v", err)
		return errors.WithStack(err)
	}
	logger.Info("Broker connected")
	return nil
}

// Stop 断开 broker。
func (l *wrapper) Stop(ctx context.Context) error {
	if Current == nil {
		return nil
	}
	if err := Current.Disconnect(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Errorf("Broker disconnect error: %v", err)
		return errors.WithStack(err)
	}
	logger.Info("Broker disconnected")
	return nil
}

// String 返回 Loader 名称。
func (l *wrapper) String() string { return "Broker" }

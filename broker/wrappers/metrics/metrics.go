// Package metrics 提供 broker.Wrapper：为 Publish/Subscribe 记录 Prometheus 指标。
//
// 用法：
//
//	b := broker.NewMemory(broker.Wrap(
//	    metrics.NewWrapper(metrics.WithAddr("localhost:9092"), metrics.WithQueue("orders")),
//	))
package metrics

import (
	"context"
	"time"

	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	kmetrics "github.com/go-kratos/kratos/v2/metrics"
	"github.com/nextmicro/next/broker"
	"github.com/nextmicro/next/pkg/metrics"
	"go.opentelemetry.io/otel/codes"
)

const namespace = "broker"

// Option 配置 metrics wrapper。
type Option func(*options)

type options struct {
	addr            string
	queue           string
	producerCounter kmetrics.Counter
	producerObserv  kmetrics.Observer
	consumerCounter kmetrics.Counter
	consumerObserv  kmetrics.Observer
}

func WithAddr(addr string) Option         { return func(o *options) { o.addr = addr } }
func WithQueue(q string) Option           { return func(o *options) { o.queue = q } }
func WithProducerCounter(c kmetrics.Counter) Option {
	return func(o *options) { o.producerCounter = c }
}
func WithProducerObserver(o2 kmetrics.Observer) Option {
	return func(o *options) { o.producerObserv = o2 }
}
func WithConsumerCounter(c kmetrics.Counter) Option {
	return func(o *options) { o.consumerCounter = c }
}
func WithConsumerObserver(o2 kmetrics.Observer) Option {
	return func(o *options) { o.consumerObserv = o2 }
}

// NewWrapper 构造 broker.Wrapper。
func NewWrapper(opts ...Option) broker.Wrapper {
	op := &options{
		producerCounter: prom.NewCounter(metrics.MessagingProducerMetricRequests),
		producerObserv:  prom.NewHistogram(metrics.MessagingProducerMetricMillisecond),
		consumerCounter: prom.NewCounter(metrics.MessagingConsumerMetricRequests),
		consumerObserv:  prom.NewHistogram(metrics.MessagingConsumerMetricMillisecond),
	}
	for _, o := range opts {
		o(op)
	}
	return func(b broker.Broker) broker.Broker {
		return &wrapper{Broker: b, opts: op}
	}
}

type wrapper struct {
	broker.Broker
	opts *options
}

func (w *wrapper) Publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	start := time.Now()
	err := w.Broker.Publish(ctx, topic, msg, opts...)
	code := codes.Ok
	if err != nil {
		code = codes.Error
	}
	w.opts.producerCounter.With(namespace, w.opts.addr, topic, code.String()).Inc()
	w.opts.producerObserv.With(namespace, w.opts.addr, topic).Observe(float64(time.Since(start).Milliseconds()))
	return err
}

func (w *wrapper) Subscribe(ctx context.Context, topic string, h broker.Handler, opts ...broker.SubscribeOption) (broker.Subscription, error) {
	wrapped := func(ctx context.Context, e broker.Event) error {
		start := time.Now()
		err := h(ctx, e)
		code := codes.Ok
		if err != nil {
			code = codes.Error
		}
		w.opts.consumerCounter.With(namespace, w.opts.addr, topic, w.opts.queue, code.String()).Inc()
		w.opts.consumerObserv.With(namespace, w.opts.addr, topic, w.opts.queue).Observe(float64(time.Since(start).Milliseconds()))
		return err
	}
	return w.Broker.Subscribe(ctx, topic, wrapped, opts...)
}

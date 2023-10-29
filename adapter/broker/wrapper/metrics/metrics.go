package metrics

import (
	"context"
	"time"

	prom "github.com/go-kratos/kratos/contrib/metrics/prometheus/v2"
	"github.com/nextmicro/next/broker"
	"github.com/nextmicro/next/pkg/metrics"
	"go.opentelemetry.io/otel/codes"
)

const namespace = "broker"

type MetricWrapper struct {
	broker.Broker

	opts *options
}

// NewMetricWrapper returns a metric wrapper for broker
func NewMetricWrapper(broker broker.Broker, opts ...Option) *MetricWrapper {
	op := &options{
		messagingProducerMetricMillisecond: prom.NewCounter(metrics.MessagingProducerMetricRequests),
		messagingProducerMetricRequests:    prom.NewHistogram(metrics.MessagingProducerMetricMillisecond),
		messagingConsumerMetricMillisecond: prom.NewCounter(metrics.MessagingConsumerMetricRequests),
		messagingConsumerMetricRequests:    prom.NewHistogram(metrics.MessagingConsumerMetricMillisecond),
	}
	for _, opt := range opts {
		opt(op)
	}

	return &MetricWrapper{
		Broker: broker,
		opts:   op,
	}
}

// Publish a message to a topic
func (w *MetricWrapper) Publish(ctx context.Context, topic string, message *broker.Message, opts ...broker.PublishOption) error {
	start := time.Now()
	err := w.Broker.Publish(ctx, topic, message, opts...)
	var code = codes.Ok
	if err != nil {
		code = codes.Error
	}

	w.opts.messagingProducerMetricMillisecond.With(namespace, w.opts.addr, topic, code.String()).Inc()
	w.opts.messagingProducerMetricRequests.With(namespace, w.opts.addr, topic).Observe(float64(time.Since(start).Milliseconds()))
	return err
}

// Subscribe to a topic
func (w *MetricWrapper) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	h := func(ctx context.Context, event broker.Event) error {
		start := time.Now()
		err := handler(ctx, event)
		var code = codes.Ok
		if err != nil {
			code = codes.Error
		}

		w.opts.messagingConsumerMetricMillisecond.With(namespace, w.opts.addr, topic, w.opts.queue, code.String()).Inc()
		w.opts.messagingConsumerMetricRequests.With(namespace, w.opts.addr, topic, w.opts.queue).Observe(float64(time.Since(start).Milliseconds()))
		return err
	}

	return w.Broker.Subscribe(topic, h, opts...)
}

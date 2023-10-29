package logging

import (
	"context"
	"time"

	"github.com/nextmicro/gokit/timex"
	"github.com/nextmicro/logger"
	"github.com/nextmicro/next/broker"
)

type wrapper struct {
	broker.Broker

	opts *options
}

// NewWrapper returns a logging wrapper for client
func NewWrapper(opts ...Option) broker.Wrapper {
	return func(b broker.Broker) broker.Broker {
		op := &options{
			SlowThreshold: 100 * time.Millisecond,
		}
		for _, opt := range opts {
			opt(op)
		}

		return &wrapper{
			Broker: b,
			opts:   op,
		}
	}
}

// Publish a message to a topic
func (w *wrapper) Publish(ctx context.Context, topic string, message *broker.Message, opts ...broker.PublishOption) error {
	start := time.Now()
	err := w.Broker.Publish(ctx, topic, message, opts...)
	duration := time.Since(start)
	fields := map[string]interface{}{
		"kind":      "messaging",
		"component": w.Broker.String(),
		"method":    topic,
		"duration":  timex.Duration(duration),
		"address":   w.opts.addr,
	}
	if err != nil {
		fields["error"] = err
	}

	log := logger.WithContext(ctx).WithFields(fields)
	if duration > w.opts.SlowThreshold {
		log.Info("[" + w.Broker.String() + "] client show")
	}

	if err != nil {
		log.Error("[" + w.Broker.String() + "] client")
	} else {
		log.Info("[" + w.Broker.String() + "] client")
	}

	return err
}

// Subscribe to a topic
func (w *wrapper) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	h := func(ctx context.Context, event broker.Event) error {
		start := time.Now()
		err := handler(ctx, event)
		duration := time.Since(start)
		fields := map[string]interface{}{
			"kind":      "messaging",
			"component": w.Broker.String(),
			"method":    topic,
			"duration":  timex.Duration(duration),
			"address":   w.opts.addr,
			"queue":     w.opts.queue,
		}
		if err != nil {
			fields["error"] = err
		}
		if w.opts.response {
			fields["header"] = event.Message().Header
			fields["body"] = string(event.Message().Body)
		}

		log := logger.WithContext(ctx).WithFields(fields)
		if duration > w.opts.SlowThreshold {
			log.Info("[" + w.Broker.String() + "] server show")
		}

		if err != nil {
			log.Error("[" + w.Broker.String() + "] server")
		} else {
			log.Info("[" + w.Broker.String() + "] server")
		}

		return err
	}

	return w.Broker.Subscribe(topic, h, opts...)
}

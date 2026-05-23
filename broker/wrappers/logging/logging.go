// Package logging 提供 broker.Wrapper：在 Publish/Subscribe 调用前后记录结构化日志。
//
// 用法：
//
//	b := broker.NewMemory(broker.Wrap(
//	    logging.NewWrapper(logging.WithAddr("localhost:9092"), logging.WithQueue("orders")),
//	))
package logging

import (
	"context"
	"time"

	"github.com/nextmicro/gokit/timex"
	"github.com/nextmicro/logger"
	"github.com/nextmicro/next/broker"
)

// Option 配置 logging wrapper 行为。
type Option func(*options)

type options struct {
	addr          string
	queue         string
	logBody       bool
	slowThreshold time.Duration
}

// WithAddr 设置 broker 地址 label。
func WithAddr(addr string) Option { return func(o *options) { o.addr = addr } }

// WithQueue 设置队列组 label。
func WithQueue(q string) Option { return func(o *options) { o.queue = q } }

// WithLogBody 是否在消费端打印消息 body（仅调试用）。
func WithLogBody(v bool) Option { return func(o *options) { o.logBody = v } }

// WithSlowThreshold 慢日志阈值。
func WithSlowThreshold(d time.Duration) Option { return func(o *options) { o.slowThreshold = d } }

// NewWrapper 构造 broker.Wrapper。
func NewWrapper(opts ...Option) broker.Wrapper {
	op := &options{slowThreshold: 100 * time.Millisecond}
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
	duration := time.Since(start)

	fields := map[string]any{
		"kind":      "publish",
		"component": "broker",
		"topic":     topic,
		"duration":  timex.Duration(duration),
		"address":   w.opts.addr,
	}
	if err != nil {
		fields["error"] = err
	}
	log := logger.WithContext(ctx).WithFields(fields)
	if duration > w.opts.slowThreshold {
		log.Info("broker publish slow")
	}
	if err != nil {
		log.Error("broker publish")
	} else {
		log.Info("broker publish")
	}
	return err
}

func (w *wrapper) Subscribe(ctx context.Context, topic string, h broker.Handler, opts ...broker.SubscribeOption) (broker.Subscription, error) {
	wrapped := func(ctx context.Context, e broker.Event) error {
		start := time.Now()
		err := h(ctx, e)
		duration := time.Since(start)

		fields := map[string]any{
			"kind":      "receive",
			"component": "broker",
			"topic":     topic,
			"duration":  timex.Duration(duration),
			"address":   w.opts.addr,
			"queue":     w.opts.queue,
		}
		if err != nil {
			fields["error"] = err
		}
		if w.opts.logBody && e.Message() != nil {
			fields["header"] = e.Message().Header
			fields["body"] = string(e.Message().Body)
		}

		log := logger.WithContext(ctx).WithFields(fields)
		if duration > w.opts.slowThreshold {
			log.Info("broker consume slow")
		}
		if err != nil {
			log.Error("broker consume")
		} else {
			log.Info("broker consume")
		}
		return err
	}
	return w.Broker.Subscribe(ctx, topic, wrapped, opts...)
}

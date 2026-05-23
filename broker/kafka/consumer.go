package kafka

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/IBM/sarama"
	tracex "github.com/nextmicro/gokit/trace"
	"github.com/nextmicro/logger"
	"github.com/nextmicro/next/broker"
	"github.com/nextmicro/next/broker/kafka/otelsarama"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// ============================================================
// Subscription 实现
// ============================================================

type kafkaSubscription struct {
	topic  string
	group  sarama.ConsumerGroup
	cancel context.CancelFunc

	once   sync.Once
	closed bool
}

func (s *kafkaSubscription) Topic() string { return s.topic }

func (s *kafkaSubscription) Close() error {
	var err error
	s.once.Do(func() {
		s.closed = true
		s.cancel()
		err = s.group.Close()
	})
	return err
}

// ============================================================
// ConsumerGroupHandler
// ============================================================

type consumerGroupHandler struct {
	ctx     context.Context
	topic   string
	handler broker.Handler
	subOpt  broker.SubscribeOptions
}

func (c *consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (c *consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (c *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-c.ctx.Done():
			return nil
		case <-session.Context().Done():
			return nil
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			if err := c.dispatch(msg, session); err != nil {
				logger.Errorf("kafka: handler error topic=%s: %v", msg.Topic, err)
			}
		}
	}
}

func (c *consumerGroupHandler) dispatch(msg *sarama.ConsumerMessage, sess sarama.ConsumerGroupSession) error {
	tr := tracex.NewTracer(trace.SpanKindConsumer)
	ctx := tr.Extract(context.Background(), otelsarama.NewConsumerMessageCarrier(msg))
	bags := baggage.FromContext(ctx)
	spanCtx := trace.SpanContextFromContext(ctx)
	ctx = baggage.ContextWithBaggage(ctx, bags)

	attrs := []attribute.KeyValue{
		semconv.MessagingOperationKey.String("process"),
	}
	ctx, span := tr.Start(
		trace.ContextWithRemoteSpanContext(ctx, spanCtx),
		fmt.Sprintf("Kafka Consumer %s", msg.Topic),
		trace.WithAttributes(attrs...),
	)
	defer span.End()

	header := make(broker.Header, len(msg.Headers))
	for _, h := range msg.Headers {
		header[string(h.Key)] = string(h.Value)
	}

	bm := &broker.Message{Header: header, Body: msg.Value}
	evt := &kafkaEvent{
		topic: strings.ReplaceAll(msg.Topic, "-", "."),
		msg:   bm,
		sess:  sess,
		raw:   msg,
	}

	err := c.handler(ctx, evt)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		_ = evt.Nack(err)
		return err
	}
	if c.subOpt.AutoAck {
		_ = evt.Ack()
	}
	return nil
}

// ============================================================
// Event 实现
// ============================================================

type kafkaEvent struct {
	topic string
	msg   *broker.Message
	sess  sarama.ConsumerGroupSession
	raw   *sarama.ConsumerMessage

	mu      sync.Mutex
	acked   bool
	nackErr error
}

func (e *kafkaEvent) Topic() string           { return e.topic }
func (e *kafkaEvent) Message() *broker.Message { return e.msg }

func (e *kafkaEvent) Ack() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.acked {
		return nil
	}
	e.acked = true
	e.sess.MarkMessage(e.raw, "")
	return nil
}

func (e *kafkaEvent) Nack(err error) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.nackErr = err
	// Kafka 没有原生 Nack 机制；保留错误记录，由调用方记录指标/转死信
	return nil
}

var (
	_ broker.Subscription = (*kafkaSubscription)(nil)
	_ broker.Event        = (*kafkaEvent)(nil)
)

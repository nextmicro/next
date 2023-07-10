package kafka

import (
	"context"
	"fmt"
	"github.com/go-volo/logger"
	"github.com/nextmicro/next/broker"
	"github.com/nextmicro/next/internal/adapter/broker/middleware"
	"strings"

	"github.com/Shopify/sarama"
	tracex "github.com/nextmicro/gokit/trace"

	"go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

type event struct {
	err                  error
	topic                string
	msg                  *broker.Message
	consumerGroup        sarama.ConsumerGroup
	consumerMessage      *sarama.ConsumerMessage
	consumerGroupSession sarama.ConsumerGroupSession
}

func (event *event) Topic() string {
	return strings.ReplaceAll(event.topic, "-", ".")
}

func (event *event) Message() *broker.Message {
	return event.msg
}

func (event *event) Ack() error {
	event.consumerGroupSession.MarkMessage(event.consumerMessage, "")
	return nil
}

func (event *event) Error() error {
	return event.err
}

type subscriber struct {
	topic         string
	cancel        context.CancelFunc
	consumerGroup sarama.ConsumerGroup
	opt           broker.SubscribeOptions
}

func (sub *subscriber) Options() broker.SubscribeOptions {
	return sub.opt
}

func (sub *subscriber) Topic() string {
	return strings.ReplaceAll(sub.topic, "-", ".")
}

func (sub *subscriber) Unsubscribe() error {
	return sub.consumerGroup.Close()
}

// consumerGroupHandler is the implementation of sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
	ctx           context.Context
	opt           broker.Options
	handler       broker.Handler
	ms            []middleware.Middleware
	subOpt        broker.SubscribeOptions
	consumerGroup sarama.ConsumerGroup
}

func (c *consumerGroupHandler) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (c *consumerGroupHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

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
			ctx, err := c.Handler(msg, session, claim)
			if err != nil {
				logger.WithContext(ctx).Errorf("broker [kafka]: subscriber , address: %s, topic: %s, error: %v", strings.Join(c.opt.Addrs, ","), msg.Topic, err)
				continue
			}
		}
	}
}

// Handler handler message
func (c *consumerGroupHandler) Handler(msg *sarama.ConsumerMessage, sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) (context.Context, error) {
	var (
		span trace.Span
		m    broker.Message
	)

	tr := tracex.NewTracer(trace.SpanKindConsumer)

	// Extract tracing info from message
	ctx := tr.Extract(context.Background(), otelsarama.NewConsumerMessageCarrier(msg))
	bags := baggage.FromContext(ctx)
	spanCtx := trace.SpanContextFromContext(ctx)
	ctx = baggage.ContextWithBaggage(ctx, bags)

	ctx, span = tr.Start(trace.ContextWithRemoteSpanContext(ctx, spanCtx), fmt.Sprintf("Kafka Consumer %s", msg.Topic), trace.WithAttributes(
		semconv.MessagingOperationProcess,
	))

	defer span.End()

	//for _, h := range msg.Headers {
	//	if h != nil && string(h.Key) == _caller {
	//		ctx = middleware.NewCallerContext(ctx,string(h.Value))
	//	}
	//}

	p := &event{msg: &m, topic: msg.Topic, consumerMessage: msg, consumerGroup: c.consumerGroup, consumerGroupSession: sess}
	errorHandler := c.opt.ErrorHandler
	if err := c.opt.Codec.Unmarshal(msg.Value, &m); err != nil {
		p.err = err
		p.msg.Body = msg.Value
		if errorHandler != nil {
			_ = errorHandler(ctx, p)
		}

		return ctx, err
	}

	h := func(ctx context.Context, topic string, req interface{}) (interface{}, error) {
		err := c.handler(ctx, p)
		return p, err
	}
	if len(c.ms) > 0 {
		h = middleware.Chain(c.ms...)(h)
	}

	_, err := h(ctx, msg.Topic, p)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		p.err = err
		if errorHandler != nil {
			_ = errorHandler(ctx, p)
		}
		return ctx, err
	}

	if c.subOpt.AutoAck {
		sess.MarkMessage(msg, "")
	}

	return ctx, nil
}
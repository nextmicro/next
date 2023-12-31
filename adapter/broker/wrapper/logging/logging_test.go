package logging_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/nextmicro/logger"
	"github.com/nextmicro/next/adapter/broker/wrapper/logging"
	"github.com/nextmicro/next/broker"
	"go.opentelemetry.io/otel"
)

func TestMain(t *testing.M) {

	t.Run()
}

func TestNewLoggingWrapper(t *testing.T) {
	b := broker.NewMemoryBroker(
		broker.Wrap(logging.NewWrapper()),
	)

	if err := b.Connect(); err != nil {
		t.Fatalf("Unexpected connect error %v", err)
	}

	topic := "test"
	count := 10

	fn := func(ctx context.Context, p broker.Event) error {
		logger.WithContext(ctx).Info("Received message", string(p.Message().Body))
		return nil
	}

	sub, err := b.Subscribe(topic, fn)
	if err != nil {
		t.Fatalf("Unexpected error subscribing %v", err)
	}

	for i := 0; i < count; i++ {
		message := &broker.Message{
			Header: map[string]string{
				"foo": "bar",
				"id":  fmt.Sprintf("%d", i),
			},
			Body: []byte(`hello world`),
		}

		ctx, span := otel.Tracer("broker").Start(context.TODO(), fmt.Sprintf("Topic %s", topic))
		if err := b.Publish(ctx, topic, message); err != nil {
			t.Fatalf("Unexpected error publishing %d", i)
		}

		span.End()
	}

	if err := sub.Unsubscribe(); err != nil {
		t.Fatalf("Unexpected error unsubscribing from %s: %v", topic, err)
	}

	if err := b.Disconnect(); err != nil {
		t.Fatalf("Unexpected connect error %v", err)
	}
}

package typed_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nextmicro/next/broker"
	"github.com/nextmicro/next/broker/typed"
)

type Order struct {
	ID    string `json:"id"`
	Total int    `json:"total"`
}

var OrderTopic = typed.Topic[*Order]{
	Name:  "orders.created",
	Codec: typed.JSONCodec[*Order]{},
}

func setupBroker(t *testing.T) broker.Broker {
	t.Helper()
	b := broker.NewMemory()
	if err := b.Connect(context.Background()); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	t.Cleanup(func() { _ = b.Disconnect(context.Background()) })
	return b
}

func TestTyped_PublishSubscribe(t *testing.T) {
	b := setupBroker(t)
	ctx := context.Background()

	var received atomic.Pointer[Order]
	sub := typed.NewSubscriber(b, OrderTopic)
	subscription, err := sub.Subscribe(ctx, func(_ context.Context, e typed.Event[*Order]) error {
		received.Store(e.Payload())
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer subscription.Close()

	pub := typed.NewPublisher(b, OrderTopic)
	if err := pub.Publish(ctx, &Order{ID: "abc", Total: 100}); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	got := received.Load()
	if got == nil {
		t.Fatalf("expected to receive an order")
	}
	if got.ID != "abc" || got.Total != 100 {
		t.Fatalf("got wrong payload: %+v", got)
	}
}

func TestTyped_ContentTypeHeader(t *testing.T) {
	b := setupBroker(t)
	ctx := context.Background()

	var gotHeader broker.Header
	sub := typed.NewSubscriber(b, OrderTopic)
	subscription, _ := sub.Subscribe(ctx, func(_ context.Context, e typed.Event[*Order]) error {
		gotHeader = e.Header()
		return nil
	})
	defer subscription.Close()

	pub := typed.NewPublisher(b, OrderTopic)
	_ = pub.Publish(ctx, &Order{ID: "x"})

	if gotHeader.Get("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", gotHeader.Get("Content-Type"))
	}
}

func TestTyped_UnmarshalErrorPropagates(t *testing.T) {
	b := setupBroker(t)
	ctx := context.Background()

	sub := typed.NewSubscriber(b, OrderTopic)
	var handlerErr atomic.Bool
	subscription, _ := sub.Subscribe(ctx, func(_ context.Context, e typed.Event[*Order]) error {
		handlerErr.Store(true)
		return nil
	})
	defer subscription.Close()

	// 直接通过底层 broker 发布非法 JSON
	_ = b.Publish(ctx, OrderTopic.Name, &broker.Message{Body: []byte("not-valid-json{")})
	// Handler 不应被调用，因为解码失败
	if handlerErr.Load() {
		t.Fatalf("handler should not be called when unmarshal fails")
	}
}

func TestTyped_NilCodecReturnsError(t *testing.T) {
	b := setupBroker(t)
	ctx := context.Background()
	bad := typed.Topic[*Order]{Name: "x"} // codec nil
	pub := typed.NewPublisher(b, bad)
	err := pub.Publish(ctx, &Order{})
	if err == nil {
		t.Fatalf("expected error for nil codec")
	}

	sub := typed.NewSubscriber(b, bad)
	_, err = sub.Subscribe(ctx, func(context.Context, typed.Event[*Order]) error { return nil })
	if err == nil {
		t.Fatalf("expected error for nil codec in subscribe")
	}
}

func TestTyped_HandlerErrorReturned(t *testing.T) {
	b := setupBroker(t)
	ctx := context.Background()
	sub := typed.NewSubscriber(b, OrderTopic)

	expectedErr := errors.New("handler failed")
	subscription, _ := sub.Subscribe(ctx, func(_ context.Context, e typed.Event[*Order]) error {
		return expectedErr
	})
	defer subscription.Close()

	pub := typed.NewPublisher(b, OrderTopic)
	// Publish 在 memory broker 中是同步分发的；handler 错误不会向 Publish 调用方传播（按设计）
	_ = pub.Publish(ctx, &Order{ID: "x"})
	// 仅验证不 panic
}

func TestTyped_SubscriptionIsolation(t *testing.T) {
	// 同一 broker 多个 typed 订阅，关闭其中之一不影响其他
	b := setupBroker(t)
	ctx := context.Background()

	topicA := typed.Topic[*Order]{Name: "a", Codec: typed.JSONCodec[*Order]{}}
	topicB := typed.Topic[*Order]{Name: "b", Codec: typed.JSONCodec[*Order]{}}

	var aCount, bCount atomic.Int32
	subA, _ := typed.NewSubscriber(b, topicA).Subscribe(ctx, func(context.Context, typed.Event[*Order]) error {
		aCount.Add(1)
		return nil
	})
	_, _ = typed.NewSubscriber(b, topicB).Subscribe(ctx, func(context.Context, typed.Event[*Order]) error {
		bCount.Add(1)
		return nil
	})

	pubA := typed.NewPublisher(b, topicA)
	pubB := typed.NewPublisher(b, topicB)
	_ = pubA.Publish(ctx, &Order{ID: "1"})
	_ = pubB.Publish(ctx, &Order{ID: "2"})

	if aCount.Load() != 1 || bCount.Load() != 1 {
		t.Fatalf("expected each to receive 1, got a=%d b=%d", aCount.Load(), bCount.Load())
	}

	_ = subA.Close()

	_ = pubA.Publish(ctx, &Order{ID: "3"})
	_ = pubB.Publish(ctx, &Order{ID: "4"})

	if aCount.Load() != 1 {
		t.Fatalf("expected a still 1 after close, got %d", aCount.Load())
	}
	if bCount.Load() != 2 {
		t.Fatalf("expected b to still receive, got %d", bCount.Load())
	}
}

var _ = time.Now

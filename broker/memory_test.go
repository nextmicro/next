package broker_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nextmicro/next/broker"
)

func TestMemoryBroker_PublishSubscribe(t *testing.T) {
	b := broker.NewMemory()
	ctx := context.Background()
	if err := b.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer b.Disconnect(ctx)

	var received atomic.Int32
	sub, err := b.Subscribe(ctx, "test", func(ctx context.Context, e broker.Event) error {
		received.Add(1)
		return nil
	})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	defer sub.Close()

	if err := b.Publish(ctx, "test", &broker.Message{Body: []byte("hello")}); err != nil {
		t.Fatalf("Publish failed: %v", err)
	}
	if received.Load() != 1 {
		t.Fatalf("expected 1 received, got %d", received.Load())
	}
}

func TestMemoryBroker_NotConnectedError(t *testing.T) {
	b := broker.NewMemory()
	ctx := context.Background()

	err := b.Publish(ctx, "x", &broker.Message{Body: []byte{}})
	if err == nil {
		t.Fatalf("expected ErrNotConnected on Publish")
	}
	_, err = b.Subscribe(ctx, "x", func(context.Context, broker.Event) error { return nil })
	if err == nil {
		t.Fatalf("expected ErrNotConnected on Subscribe")
	}
}

func TestMemoryBroker_CloseSubscription(t *testing.T) {
	b := broker.NewMemory()
	ctx := context.Background()
	_ = b.Connect(ctx)
	defer b.Disconnect(ctx)

	var received atomic.Int32
	sub, _ := b.Subscribe(ctx, "t", func(context.Context, broker.Event) error {
		received.Add(1)
		return nil
	})

	_ = b.Publish(ctx, "t", &broker.Message{Body: []byte("a")})
	if received.Load() != 1 {
		t.Fatalf("expected 1 received before close, got %d", received.Load())
	}

	_ = sub.Close()
	_ = b.Publish(ctx, "t", &broker.Message{Body: []byte("b")})
	if received.Load() != 1 {
		t.Fatalf("expected still 1 after close, got %d", received.Load())
	}

	// Close 应是幂等的
	if err := sub.Close(); err != nil {
		t.Fatalf("Close should be idempotent, got %v", err)
	}
}

func TestMemoryBroker_QueueGroupDelivery(t *testing.T) {
	b := broker.NewMemory()
	ctx := context.Background()
	_ = b.Connect(ctx)
	defer b.Disconnect(ctx)

	var groupAReceived atomic.Int32
	var groupBReceived atomic.Int32
	var noGroupReceived atomic.Int32

	// 同组 A 的两个订阅者：只有一个会收到
	_, _ = b.Subscribe(ctx, "topic",
		func(context.Context, broker.Event) error { groupAReceived.Add(1); return nil },
		broker.WithSubscribeQueue("groupA"))
	_, _ = b.Subscribe(ctx, "topic",
		func(context.Context, broker.Event) error { groupAReceived.Add(1); return nil },
		broker.WithSubscribeQueue("groupA"))

	// 同组 B 的一个订阅者
	_, _ = b.Subscribe(ctx, "topic",
		func(context.Context, broker.Event) error { groupBReceived.Add(1); return nil },
		broker.WithSubscribeQueue("groupB"))

	// 无队列组的订阅者：每次都收到
	_, _ = b.Subscribe(ctx, "topic",
		func(context.Context, broker.Event) error { noGroupReceived.Add(1); return nil })

	_ = b.Publish(ctx, "topic", &broker.Message{Body: []byte("x")})

	if groupAReceived.Load() != 1 {
		t.Fatalf("groupA should receive 1, got %d", groupAReceived.Load())
	}
	if groupBReceived.Load() != 1 {
		t.Fatalf("groupB should receive 1, got %d", groupBReceived.Load())
	}
	if noGroupReceived.Load() != 1 {
		t.Fatalf("no-group should receive 1, got %d", noGroupReceived.Load())
	}
}

func TestMemoryBroker_PublishHeaders(t *testing.T) {
	b := broker.NewMemory()
	ctx := context.Background()
	_ = b.Connect(ctx)
	defer b.Disconnect(ctx)

	var gotHeader broker.Header
	_, _ = b.Subscribe(ctx, "t", func(_ context.Context, e broker.Event) error {
		gotHeader = e.Message().Header.Clone()
		return nil
	})

	_ = b.Publish(ctx, "t", &broker.Message{Body: []byte("x")},
		broker.WithPublishHeader("trace-id", "abc"),
		broker.WithPublishHeader("user-id", "42"))

	if gotHeader.Get("trace-id") != "abc" {
		t.Fatalf("expected trace-id=abc, got %q", gotHeader.Get("trace-id"))
	}
	if gotHeader.Get("user-id") != "42" {
		t.Fatalf("expected user-id=42, got %q", gotHeader.Get("user-id"))
	}
}

func TestMemoryBroker_DisconnectClosesAllSubs(t *testing.T) {
	b := broker.NewMemory()
	ctx := context.Background()
	_ = b.Connect(ctx)

	var received atomic.Int32
	_, _ = b.Subscribe(ctx, "t", func(context.Context, broker.Event) error {
		received.Add(1)
		return nil
	})
	_ = b.Disconnect(ctx)

	err := b.Publish(ctx, "t", &broker.Message{Body: []byte("x")})
	if err == nil {
		t.Fatalf("expected ErrNotConnected after Disconnect")
	}
	if received.Load() != 0 {
		t.Fatalf("expected 0 after disconnect, got %d", received.Load())
	}
}

func TestWrapper_AppliedInOrder(t *testing.T) {
	var order []string
	var mu sync.Mutex

	tag := func(name string) broker.Wrapper {
		return func(b broker.Broker) broker.Broker {
			return &tagBroker{Broker: b, name: name, order: &order, mu: &mu}
		}
	}

	b := broker.NewMemory(broker.Wrap(tag("outer"), tag("inner")))
	ctx := context.Background()
	_ = b.Connect(ctx)
	defer b.Disconnect(ctx)

	_, _ = b.Subscribe(ctx, "t", func(context.Context, broker.Event) error { return nil })
	_ = b.Publish(ctx, "t", &broker.Message{Body: []byte("x")})

	mu.Lock()
	defer mu.Unlock()
	// 顺序应为：outer.Publish -> inner.Publish -> memory.Publish
	if len(order) < 2 || order[0] != "outer" || order[1] != "inner" {
		t.Fatalf("expected outer->inner, got %v", order)
	}
}

type tagBroker struct {
	broker.Broker
	name  string
	order *[]string
	mu    *sync.Mutex
}

func (t *tagBroker) Publish(ctx context.Context, topic string, m *broker.Message, opts ...broker.PublishOption) error {
	t.mu.Lock()
	*t.order = append(*t.order, t.name)
	t.mu.Unlock()
	return t.Broker.Publish(ctx, topic, m, opts...)
}

// 防止 time 包未被使用
var _ = time.Now

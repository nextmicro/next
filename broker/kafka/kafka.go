// Package kafka 提供基于 IBM/sarama 的 Kafka broker.Broker 实现。
//
// 设计原则：
//   - 仅实现底层 broker.Broker 接口（Connect/Disconnect/Publish/Subscribe）
//   - 不做 codec marshal/unmarshal，由上层 typed.Publisher[T]/Subscriber[T] 负责
//   - Header 通过 sarama record headers 透传
//   - 通过 ConfigOption / Key 等扩展点向 sarama 注入额外配置
package kafka

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/IBM/sarama"
	tracex "github.com/nextmicro/gokit/trace"
	"github.com/nextmicro/logger"
	"github.com/nextmicro/next/broker"
	"github.com/nextmicro/next/broker/kafka/otelsarama"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const namespace = "kafka"

// ============================================================
// 选项扩展（kafka 特有，超出 broker.Option 通用范围）
// ============================================================

type (
	producerConfigKey struct{}
	consumerConfigKey struct{}
	messageKeyHeader  = string
)

// MessageKeyHeader 是发布消息时用于设置 kafka 分区键的 Header key。
// 通过 broker.WithPublishHeader(MessageKeyHeader, "user-1") 设置。
const MessageKeyHeader = "kafka-message-key"

// ProducerConfig 注入自定义 sarama.Config（producer 端）。
// 通过 broker.Option → broker.Options.Context 携带。
func ProducerConfig(c *sarama.Config) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, producerConfigKey{}, c)
	}
}

// ConsumerConfig 注入自定义 sarama.Config（consumer 端）。
func ConsumerConfig(c *sarama.Config) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, consumerConfigKey{}, c)
	}
}

// ============================================================
// Broker 实现
// ============================================================

// Kafka 是基于 sarama 的 broker.Broker 实现。
type Kafka struct {
	opts broker.Options

	rootCtx context.Context
	cancel  context.CancelFunc

	mu           sync.Mutex
	connected    bool
	closed       atomic.Bool
	syncProducer sarama.SyncProducer
	clients      []sarama.Client
}

// New 构造 Kafka broker。
func New(opts ...broker.Option) broker.Broker {
	o := broker.NewOptions(opts...)
	if len(o.Addrs) == 0 {
		o.Addrs = []string{"127.0.0.1:9092"}
	}
	rootCtx, cancel := context.WithCancel(o.Context)
	k := &Kafka{opts: o, rootCtx: rootCtx, cancel: cancel}

	var br broker.Broker = k
	for i := len(o.Wrappers) - 1; i >= 0; i-- {
		br = o.Wrappers[i](br)
	}
	return br
}

// Connect 实现 broker.Broker.Connect。
func (k *Kafka) Connect(_ context.Context) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.connected {
		return nil
	}

	cfg := k.producerConfig()
	client, err := sarama.NewClient(k.opts.Addrs, cfg)
	if err != nil {
		return fmt.Errorf("kafka: new client: %w", err)
	}
	producer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		return fmt.Errorf("kafka: new producer: %w", err)
	}

	producer = otelsarama.WrapSyncProducer(cfg, producer)

	k.syncProducer = producer
	k.clients = append(k.clients, client)
	k.connected = true
	logger.Infof("broker [%s] connected, addrs=%s", namespace, strings.Join(k.opts.Addrs, ","))
	return nil
}

// Disconnect 实现 broker.Broker.Disconnect。
func (k *Kafka) Disconnect(_ context.Context) error {
	if !k.closed.CompareAndSwap(false, true) {
		return nil
	}
	k.cancel()

	k.mu.Lock()
	defer k.mu.Unlock()
	if !k.connected {
		return nil
	}
	if k.syncProducer != nil {
		_ = k.syncProducer.Close()
	}
	for _, c := range k.clients {
		_ = c.Close()
	}
	k.connected = false
	logger.Infof("broker [%s] disconnected", namespace)
	return nil
}

// Publish 实现 broker.Broker.Publish。
// msg.Header 中的所有 key/value 会作为 record headers 发送；
// "kafka-message-key" 特殊键被用作分区键（如果存在）。
func (k *Kafka) Publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	if k.closed.Load() {
		return errors.New("kafka: broker closed")
	}
	if msg == nil {
		return errors.New("kafka: nil message")
	}

	po := broker.NewPublishOptions(opts...)
	headers := broker.Header{}
	if msg.Header != nil {
		for k2, v := range msg.Header {
			headers[k2] = v
		}
	}
	for k2, v := range po.Header {
		headers[k2] = v
	}

	saramaTopic := strings.ReplaceAll(topic, ".", "-")
	pm := &sarama.ProducerMessage{
		Topic: saramaTopic,
		Value: sarama.ByteEncoder(msg.Body),
	}
	if key, ok := headers[MessageKeyHeader]; ok && key != "" {
		pm.Key = sarama.StringEncoder(key)
		delete(headers, MessageKeyHeader)
	}
	for hk, hv := range headers {
		pm.Headers = append(pm.Headers, sarama.RecordHeader{Key: []byte(hk), Value: []byte(hv)})
	}

	tr := tracex.NewTracer(trace.SpanKindProducer)
	var span trace.Span
	ctx, span = tr.Start(ctx, fmt.Sprintf("KF Producer %s", pm.Topic))
	defer span.End()
	tr.Inject(ctx, otelsarama.NewProducerMessageCarrier(pm))

	_, _, err := k.syncProducer.SendMessage(pm)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

// Subscribe 实现 broker.Broker.Subscribe。
// 通过 sarama ConsumerGroup 订阅；返回的 Subscription.Close 仅关闭该订阅。
func (k *Kafka) Subscribe(ctx context.Context, topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscription, error) {
	if k.closed.Load() {
		return nil, errors.New("kafka: broker closed")
	}
	so := broker.NewSubscribeOptions(opts...)
	saramaTopic := strings.ReplaceAll(topic, ".", "-")

	client, err := k.newConsumerClient()
	if err != nil {
		return nil, err
	}
	group, err := sarama.NewConsumerGroupFromClient(so.Queue, client)
	if err != nil {
		return nil, err
	}

	subCtx, cancel := context.WithCancel(k.rootCtx)
	cgh := &consumerGroupHandler{
		ctx:     subCtx,
		topic:   topic,
		handler: handler,
		subOpt:  so,
	}
	wrappedHandler := otelsarama.WrapConsumerGroupHandler(cgh)

	go func() {
		topics := []string{saramaTopic}
		for {
			select {
			case <-subCtx.Done():
				return
			default:
				if err := group.Consume(subCtx, topics, wrappedHandler); err != nil {
					if errors.Is(err, sarama.ErrClosedConsumerGroup) || errors.Is(err, sarama.ErrClosedClient) {
						return
					}
					logger.Errorf("kafka: consume error: %v", err)
				}
			}
		}
	}()

	return &kafkaSubscription{topic: topic, group: group, cancel: cancel}, nil
}

// ============================================================
// 内部辅助
// ============================================================

func (k *Kafka) producerConfig() *sarama.Config {
	if v, ok := k.opts.Context.Value(producerConfigKey{}).(*sarama.Config); ok {
		return v
	}
	c := sarama.NewConfig()
	c.Version = sarama.V3_5_1_0
	c.Producer.Return.Errors = true
	c.Producer.Return.Successes = true
	c.Producer.RequiredAcks = sarama.WaitForAll
	return c
}

func (k *Kafka) consumerConfig() *sarama.Config {
	if v, ok := k.opts.Context.Value(consumerConfigKey{}).(*sarama.Config); ok {
		return v
	}
	c := sarama.NewConfig()
	c.Version = sarama.V3_2_0_0
	c.Consumer.Return.Errors = true
	c.Consumer.Offsets.Initial = sarama.OffsetNewest
	return c
}

func (k *Kafka) newConsumerClient() (sarama.Client, error) {
	cfg := k.consumerConfig()
	client, err := sarama.NewClient(k.opts.Addrs, cfg)
	if err != nil {
		return nil, err
	}
	k.mu.Lock()
	k.clients = append(k.clients, client)
	k.mu.Unlock()
	return client, nil
}

// ensure interface
var _ broker.Broker = (*Kafka)(nil)

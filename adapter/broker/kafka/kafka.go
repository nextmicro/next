package kafka

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/nextmicro/next/adapter/broker/middleware"

	"github.com/nextmicro/logger"
	adapter "github.com/nextmicro/next/adapter/logger/log"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/Shopify/sarama"
	_ "github.com/go-kratos/kratos/v2/encoding/proto"
	tracex "github.com/nextmicro/gokit/trace"
	b "github.com/nextmicro/next/broker"
	"go.opentelemetry.io/contrib/instrumentation/github.com/Shopify/sarama/otelsarama"
)

func init() {
	log := adapter.New(logger.DefaultLogger)
	sarama.Logger = log
	sarama.DebugLogger = log
}

type Kafka struct {
	closed       int32
	connected    bool
	mutex        sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	clients      []sarama.Client
	publishMs    []middleware.Middleware
	subscribeMs  []middleware.Middleware
	syncProducer sarama.SyncProducer
	opt          b.Options
}

func New(opts ...b.Option) b.Broker {
	opt := b.Options{
		Addrs:   []string{"127.0.0.1:9092"},
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&opt)
	}

	var (
		publishMs   []middleware.Middleware
		subscribeMs []middleware.Middleware
	)

	ctx, cancel := context.WithCancel(opt.Context)
	return &Kafka{
		opt:         opt,
		ctx:         ctx,
		cancel:      cancel,
		publishMs:   publishMs,
		subscribeMs: subscribeMs,
	}
}

func (broker *Kafka) Init(opts ...b.Option) error {
	for _, o := range opts {
		o(&broker.opt)
	}
	return nil
}

func (broker *Kafka) Options() b.Options {
	return broker.opt
}

func (broker *Kafka) Address() string {
	return strings.Join(broker.opt.Addrs, ",")
}

func (broker *Kafka) markClosed() {
	atomic.StoreInt32(&broker.closed, 1)
}

func (broker *Kafka) isClosed() bool {
	return atomic.LoadInt32(&broker.closed) != 0
}

func (broker *Kafka) getProducerConfig() *sarama.Config {
	if c, ok := broker.opt.Context.Value(publishConfigKey{}).(*sarama.Config); ok {
		return c
	}
	return sarama.NewConfig()
}

func (broker *Kafka) Connect() error {
	if broker.connected {
		return nil
	}

	broker.mutex.Lock()
	if broker.syncProducer != nil {
		broker.mutex.Unlock()
		return nil
	}
	broker.mutex.Unlock()

	cfg := broker.getProducerConfig()
	cfg.Version = sarama.V3_2_0_0
	cfg.Producer.Return.Errors = true
	cfg.Producer.Return.Successes = true
	cfg.Producer.RequiredAcks = sarama.WaitForAll

	cfg.ClientID = broker.opt.SubscribeOptions.Queue

	logger.Infof("broker [%s] queue: %s", broker.String(), cfg.ClientID)

	client, err := sarama.NewClient(broker.opt.Addrs, cfg)
	if err != nil {
		return fmt.Errorf("broker: kafak error: %v", err)
	}

	producer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		return err
	}

	// Wrap instrumentation
	producer = otelsarama.WrapSyncProducer(cfg, producer)

	broker.mutex.Lock()
	broker.syncProducer = producer
	broker.connected = true
	defer broker.mutex.Unlock()

	return nil
}

func (broker *Kafka) Disconnect() error {
	if broker.isClosed() {
		return nil
	}

	broker.cancel()

	broker.mutex.Lock()
	defer broker.mutex.Unlock()

	if !broker.connected {
		return nil
	}

	for _, client := range broker.clients {
		client.Close()
	}

	broker.syncProducer.Close()

	broker.connected = false
	broker.markClosed()

	logger.Infof("broker [%s] closed success", b.String())
	return nil
}

func (broker *Kafka) Publish(ctx context.Context, topic string, msg *b.Message, opts ...b.PublishOption) error {
	if broker.isClosed() {
		return io.EOF
	}

	var (
		ms  []middleware.Middleware
		opt = b.PublishOptions{
			Context: context.Background(),
		}
	)
	for _, o := range opts {
		o(&opt)
	}

	ms = append(ms, broker.publishMs...)
	topic = strings.ReplaceAll(topic, ".", "-")
	bytes, err := broker.opt.Codec.Marshal(msg)
	if err != nil {
		return err
	}

	message := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(bytes),
	}

	if value, ok := opt.Context.Value(publishMessageKey{}).(string); ok {
		message.Key = sarama.ByteEncoder(value)
	}

	tr := tracex.NewTracer(trace.SpanKindProducer)
	ctx, span := tr.Start(opt.Context, fmt.Sprintf("KF Producer %s", message.Topic))
	defer span.End()

	tr.Inject(ctx, otelsarama.NewProducerMessageCarrier(message))

	h := func(ctx context.Context, topic string, req interface{}) (interface{}, error) {
		partition, offset, err := broker.syncProducer.SendMessage(req.(*sarama.ProducerMessage))
		return &SendMessageResponse{
			partition: partition,
			offset:    offset,
		}, err
	}

	if userMs, ok := opt.Context.Value(publishMiddlewaresKey{}).([]middleware.Middleware); ok {
		ms = append(ms, userMs...)
	}
	if len(ms) > 0 {
		h = middleware.Chain(ms...)(h)
	}

	_, err = h(opt.Context, topic, message)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (broker *Kafka) getConsumerConfig() *sarama.Config {
	if c, ok := broker.opt.Context.Value(subscribeConfigKey{}).(*sarama.Config); ok {
		return c
	}

	cfg := sarama.NewConfig()
	cfg.Version = sarama.V3_2_0_0
	cfg.Consumer.Return.Errors = true
	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	return cfg
}

func (broker *Kafka) getSaramaClusterClient() (sarama.Client, error) {
	config := broker.getConsumerConfig()
	client, err := sarama.NewClient(broker.opt.Addrs, config)
	if err != nil {
		return nil, err
	}
	broker.mutex.Lock()
	broker.clients = append(broker.clients, client)
	broker.mutex.Unlock()
	return client, nil
}

func (broker *Kafka) Subscribe(topic string, h b.Handler, opts ...b.SubscribeOption) (b.Subscriber, error) {
	if broker.isClosed() {
		return nil, io.EOF
	}

	var (
		ms  []middleware.Middleware
		opt = b.SubscribeOptions{
			Queue:   broker.opt.SubscribeOptions.Queue,
			AutoAck: broker.opt.SubscribeOptions.AutoAck,
			Context: broker.opt.SubscribeOptions.Context,
		}
	)
	for _, o := range opts {
		o(&opt)
	}

	logger.Infof("broker [%s] queue: %s Subscribe topic: %s", broker.String(), opt.Queue, topic)

	ms = append(ms, broker.subscribeMs...)
	topic = strings.ReplaceAll(topic, ".", "-")
	// we need to create a new client per consumer
	client, err := broker.getSaramaClusterClient()
	if err != nil {
		return nil, err
	}

	consumerGroup, err := sarama.NewConsumerGroupFromClient(opt.Queue, client)
	if err != nil {
		return nil, err
	}
	if userMs, ok := opt.Context.Value(subscribeMiddlewaresKey{}).([]middleware.Middleware); ok {
		ms = append(ms, userMs...)
	}

	consumerGroupHandler := &consumerGroupHandler{
		handler:       h,
		subOpt:        opt,
		opt:           broker.opt,
		ms:            ms,
		consumerGroup: consumerGroup,
		ctx:           broker.ctx,
	}

	topics := []string{topic}
	handler := otelsarama.WrapConsumerGroupHandler(consumerGroupHandler)

	go func() {
		for {
			select {
			case <-broker.ctx.Done():
				goto close
			case err, ok := <-consumerGroup.Errors():
				if !ok {
					goto close
				}

				logger.Errorf("consumer error: %v", err)
			default:
				err = consumerGroup.Consume(broker.ctx, topics, handler)
				if errors.Is(err, sarama.ErrClosedConsumerGroup) || errors.Is(err, sarama.ErrClosedClient) {
					goto close
				} else if err != nil {
					logger.Errorf("Error from consumer: %v", err)
				}
			}
		}

	close:
	}()

	return &subscriber{
		topic:         topic,
		opt:           opt,
		cancel:        broker.cancel,
		consumerGroup: consumerGroup,
	}, nil
}

func (broker *Kafka) String() string {
	return namespace
}

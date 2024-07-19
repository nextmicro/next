package otelnsq

import (
	"time"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel/propagation"
)

var _ propagation.TextMapCarrier = (*ProducerMessageCarrier)(nil)
var _ propagation.TextMapCarrier = (*ConsumerMessageCarrier)(nil)

// RecordHeader stores key and value for a record header
type RecordHeader struct {
	Key   []byte
	Value []byte
}

// ProducerMessage is the collection of elements passed to the Producer in order to send a message.
type ProducerMessage struct {
	Topic string        // The Kafka topic for this message.
	Delay time.Duration // The delay to wait before sending the message.
	Body  []byte        // The message contents.
	// The headers are key-value pairs that are transparently passed
	// by Kafka between producers and consumers.
	Headers []RecordHeader
}

// ProducerMessageCarrier injects and extracts traces from a sarama.ProducerMessage.
type ProducerMessageCarrier struct {
	msg *ProducerMessage
}

// NewProducerMessageCarrier creates a new ProducerMessageCarrier.
func NewProducerMessageCarrier(msg *ProducerMessage) ProducerMessageCarrier {
	return ProducerMessageCarrier{msg: msg}
}

// Get retrieves a single value for a given key.
func (c ProducerMessageCarrier) Get(key string) string {
	for _, h := range c.msg.Headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

// Set sets a header.
func (c ProducerMessageCarrier) Set(key, val string) {
	// Ensure uniqueness of keys
	for i := 0; i < len(c.msg.Headers); i++ {
		if string(c.msg.Headers[i].Key) == key {
			c.msg.Headers = append(c.msg.Headers[:i], c.msg.Headers[i+1:]...)
			i--
		}
	}
	c.msg.Headers = append(c.msg.Headers, RecordHeader{
		Key:   []byte(key),
		Value: []byte(val),
	})
}

// Keys returns a slice of all key identifiers in the carrier.
func (c ProducerMessageCarrier) Keys() []string {
	out := make([]string, len(c.msg.Headers))
	for i, h := range c.msg.Headers {
		out[i] = string(h.Key)
	}
	return out
}

// ConsumerMessageCarrier injects and extracts traces from a sarama.ConsumerMessage.
type ConsumerMessageCarrier struct {
	msg *sarama.ConsumerMessage
}

// NewConsumerMessageCarrier creates a new ConsumerMessageCarrier.
func NewConsumerMessageCarrier(msg *sarama.ConsumerMessage) ConsumerMessageCarrier {
	return ConsumerMessageCarrier{msg: msg}
}

// Get retrieves a single value for a given key.
func (c ConsumerMessageCarrier) Get(key string) string {
	for _, h := range c.msg.Headers {
		if h != nil && string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

// Set sets a header.
func (c ConsumerMessageCarrier) Set(key, val string) {
	// Ensure uniqueness of keys
	for i := 0; i < len(c.msg.Headers); i++ {
		if c.msg.Headers[i] != nil && string(c.msg.Headers[i].Key) == key {
			c.msg.Headers = append(c.msg.Headers[:i], c.msg.Headers[i+1:]...)
			i--
		}
	}
	c.msg.Headers = append(c.msg.Headers, &sarama.RecordHeader{
		Key:   []byte(key),
		Value: []byte(val),
	})
}

// Keys returns a slice of all key identifiers in the carrier.
func (c ConsumerMessageCarrier) Keys() []string {
	out := make([]string, len(c.msg.Headers))
	for i, h := range c.msg.Headers {
		out[i] = string(h.Key)
	}
	return out
}

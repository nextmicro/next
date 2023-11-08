package kafka

import (
	"context"

	"github.com/nextmicro/next/broker"

	"github.com/IBM/sarama"
)

const (
	namespace = "kafka"
)

type (
	publishConfigKey    struct{}
	subscribeConfigKey  struct{}
	publishMessageKey   struct{}
	SendMessageResponse struct {
		partition int32
		offset    int64
	}
)

// setBrokerOption returns a function to setup a context with given value
func setBrokerOption(k, v interface{}) broker.Option {
	return func(o *broker.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

// setPublishOption returns a function to setup a context with given value
func setPublishOption(k, v interface{}) broker.PublishOption {
	return func(o *broker.PublishOptions) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, k, v)
	}
}

func PublishConfig(c *sarama.Config) broker.Option {
	return setBrokerOption(publishConfigKey{}, c)
}

func SubscribeConfig(c *sarama.Config) broker.Option {
	return setBrokerOption(subscribeConfigKey{}, c)
}

// Key The partitioning key for this message. Pre-existing Encoders include
// StringEncoder.
func Key(key string) broker.PublishOption {
	return setPublishOption(publishMessageKey{}, key)
}

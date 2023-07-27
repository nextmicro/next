package kafka

import (
	"context"

	"github.com/nextmicro/next/adapter/broker/middleware"
	"github.com/nextmicro/next/broker"

	"github.com/Shopify/sarama"
)

const (
	namespace = "kafka"
)

type (
	serviceNameKey          struct{}
	publishConfigKey        struct{}
	subscribeConfigKey      struct{}
	publishMessageKey       struct{}
	publishMiddlewaresKey   struct{}
	subscribeMiddlewaresKey struct{}
	SendMessageResponse     struct {
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

// setSubscribeOption returns a function to setup a context with given value
func setSubscribeOption(k, v interface{}) broker.SubscribeOption {
	return func(o *broker.SubscribeOptions) {
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

func PublishMiddleware(ms ...middleware.Middleware) broker.PublishOption {
	return setPublishOption(publishMiddlewaresKey{}, ms)
}

func SubscribeMiddleware(ms ...middleware.Middleware) broker.SubscribeOption {
	return setSubscribeOption(subscribeMiddlewaresKey{}, ms)
}

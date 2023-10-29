package metrics

import (
	"github.com/go-kratos/kratos/v2/metrics"
)

// Option is metrics option.
type Option func(options *options)

type options struct {
	addr                               string
	queue                              string
	messagingProducerMetricMillisecond metrics.Counter
	messagingProducerMetricRequests    metrics.Observer
	messagingConsumerMetricMillisecond metrics.Counter
	messagingConsumerMetricRequests    metrics.Observer
}

// WithQueue with queue label.
func WithQueue(queue string) Option {
	return func(o *options) {
		o.queue = queue
	}
}

// WithAddr with addr label.
func WithAddr(address string) Option {
	return func(o *options) {
		o.addr = address
	}
}

// WithProducerCounter with messaging producer metric millisecond.
func WithProducerCounter(counter metrics.Counter) Option {
	return func(o *options) {
		o.messagingProducerMetricMillisecond = counter
	}
}

// WithProducerObserver with messaging producer metric requests.
func WithProducerObserver(observer metrics.Observer) Option {
	return func(o *options) {
		o.messagingProducerMetricRequests = observer
	}
}

// WithConsumerCounter with messaging consumer metric millisecond.
func WithConsumerCounter(counter metrics.Counter) Option {
	return func(o *options) {
		o.messagingConsumerMetricMillisecond = counter
	}
}

// WithConsumerObserver with messaging consumer metric requests.
func WithConsumerObserver(observer metrics.Observer) Option {
	return func(o *options) {
		o.messagingConsumerMetricRequests = observer
	}
}

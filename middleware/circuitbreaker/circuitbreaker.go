package circuitbreaker

import (
	"context"
	"math/rand"
	"sync"

	"github.com/go-kratos/aegis/circuitbreaker"
	"github.com/go-kratos/aegis/circuitbreaker/sre"
	log "github.com/nextmicro/logger"
	config "github.com/nextmicro/next/api/config/v1"
	v1 "github.com/nextmicro/next/api/middleware/circuitbreaker/v1"
	chain "github.com/nextmicro/next/middleware"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
)

// ErrNotAllowed is request failed due to circuit breaker triggered.
var ErrNotAllowed = errors.New(503, "CIRCUITBREAKER", "request failed due to circuit breaker triggered")

func init() {
	chain.Register("client.circuitbreaker", injection)
}

type ratioTrigger struct {
	*v1.CircuitBreaker_Ratio
	lock sync.Mutex
	rand *rand.Rand
}

func newRatioTrigger(in *v1.CircuitBreaker_Ratio) *ratioTrigger {
	return &ratioTrigger{
		CircuitBreaker_Ratio: in,
		rand:                 rand.New(rand.NewSource(rand.Int63())),
	}
}

func (r *ratioTrigger) Allow() error {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.rand.Int63n(10000) < r.Ratio {
		return nil
	}
	return circuitbreaker.ErrNotAllowed
}
func (*ratioTrigger) MarkSuccess() {}
func (*ratioTrigger) MarkFailed()  {}

type nopTrigger struct{}

func (nopTrigger) Allow() error { return nil }
func (nopTrigger) MarkSuccess() {}
func (nopTrigger) MarkFailed()  {}

func makeBreakerTrigger(in *v1.CircuitBreaker) circuitbreaker.CircuitBreaker {
	switch trigger := in.Trigger.(type) {
	case *v1.CircuitBreaker_SuccessRatio:
		var opts []sre.Option
		if trigger.SuccessRatio.Bucket != 0 {
			opts = append(opts, sre.WithBucket(int(trigger.SuccessRatio.Bucket)))
		}
		if trigger.SuccessRatio.Request != 0 {
			opts = append(opts, sre.WithRequest(int64(trigger.SuccessRatio.Request)))
		}
		if trigger.SuccessRatio.Success != 0 {
			opts = append(opts, sre.WithSuccess(trigger.SuccessRatio.Success))
		}
		if trigger.SuccessRatio.Window != nil {
			opts = append(opts, sre.WithWindow(trigger.SuccessRatio.Window.AsDuration()))
		}
		return sre.NewBreaker(opts...)
	case *v1.CircuitBreaker_Ratio:
		return newRatioTrigger(trigger)
	default:
		log.Warnf("Unrecoginzed circuit breaker trigger: %+v", trigger)
		return nopTrigger{}
	}
}

func injection(c *config.Middleware) (middleware.Middleware, error) {
	options := &v1.CircuitBreaker{}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, options, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}

	opts := make([]Option, 0, 1)
	breaker := makeBreakerTrigger(options)
	if breaker != nil {
		opts = append(opts, WithCircuitBreaker(breaker))
	}

	return Client(opts...), nil
}

// Option represents options update func
type Option func(*Options)

// Options represents hystrix client wrapper options
type Options struct {
	breaker circuitbreaker.CircuitBreaker
}

func WithCircuitBreaker(c circuitbreaker.CircuitBreaker) Option {
	return func(o *Options) {
		o.breaker = c
	}
}

// Client circuitbreaker middleware will return errBreakerTriggered when the circuit
// breaker is triggered and the request is rejected directly.
func Client(opts ...Option) middleware.Middleware {
	options := Options{
		breaker: sre.NewBreaker(),
	}
	for _, o := range opts {
		o(&options)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if err := options.breaker.Allow(); err != nil {
				// rejected
				// NOTE: when client reject requests locally,
				// continue to add counter let the drop ratio higher.
				options.breaker.MarkFailed()
				return nil, ErrNotAllowed
			}
			// allowed
			reply, err := handler(ctx, req)
			if err != nil && (errors.IsInternalServer(err) || errors.IsServiceUnavailable(err) || errors.IsGatewayTimeout(err)) {
				options.breaker.MarkFailed()
			} else {
				options.breaker.MarkSuccess()
			}
			return reply, err
		}
	}
}

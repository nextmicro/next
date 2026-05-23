package circuitbreaker

import (
	"math/rand"
	"sync"

	"github.com/go-kratos/aegis/circuitbreaker"
	"github.com/go-kratos/aegis/circuitbreaker/sre"
	log "github.com/nextmicro/logger"
	v1 "github.com/nextmicro/next/api/middleware/circuitbreaker/v1"
)

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

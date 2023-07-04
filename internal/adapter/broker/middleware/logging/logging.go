package logging

import (
	"context"
	"github.com/go-volo/logger"
	"github.com/nextmicro/gokit/timex"
	"github.com/nextmicro/next/internal/adapter/broker/middleware"
	"time"
)

func Client(opts ...Option) middleware.Middleware {
	op := &options{
		SlowThreshold: 100 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(op)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, topic string, req interface{}) (reply interface{}, err error) {
			var (
				start = time.Now()
			)

			reply, err = handler(ctx, topic, req)
			if err != nil {
				return
			}

			duration := time.Since(start)
			fields := map[string]interface{}{
				"kind":      "messaging",
				"component": op.namespace,
				"method":    topic,
				"duration":  timex.Duration(duration),
			}
			if op.request {
				fields["req"] = req
			}
			if err != nil {
				fields["error"] = err
			}

			log := logger.WithContext(ctx).WithFields(fields)
			if duration > op.SlowThreshold {
				log.Info("[" + op.namespace + "] client show")
			}

			if err != nil {
				log.Error("[" + op.namespace + "] client")
			} else {
				log.Info("[" + op.namespace + "] client")
			}

			return
		}
	}
}

func Server(opts ...Option) middleware.Middleware {
	op := &options{
		SlowThreshold: 100 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(op)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, topic string, req interface{}) (reply interface{}, err error) {
			var (
				start = time.Now()
			)

			reply, err = handler(ctx, topic, req)
			if err != nil {
				return
			}

			duration := time.Since(start)
			fields := map[string]interface{}{
				"kind":      "messaging",
				"component": op.namespace,
				"method":    topic,
				"duration":  timex.Duration(duration),
			}
			if op.request {
				fields["req"] = req
			}
			if err != nil {
				fields["error"] = err
			}

			log := logger.WithContext(ctx).WithFields(fields)
			if duration > op.SlowThreshold {
				log.Info("[" + op.namespace + "] server show")
			}

			if err != nil {
				log.Error("[" + op.namespace + "] server")
			} else {
				log.Info("[" + op.namespace + "] server")
			}

			return
		}
	}
}

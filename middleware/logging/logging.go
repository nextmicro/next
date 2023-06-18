package logging

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-volo/logger"
	"github.com/nextmicro/gokit/timex"
)

// Redacter defines how to log an object
type Redacter interface {
	Redact() string
}

// extractError returns the string of the error
func extractError(err error) string {
	if err == nil {
		return ""
	}

	return fmt.Sprintf("%+v", err)
}

// extractArgs returns the string of the req
func extractArgs(req interface{}) string {
	if redacter, ok := req.(Redacter); ok {
		return redacter.Redact()
	}
	if stringer, ok := req.(fmt.Stringer); ok {
		return stringer.String()
	}
	return fmt.Sprintf("%+v", req)
}

// mergeFields merges the fields
func mergeFields(fields map[string]interface{}, m map[string]string) map[string]interface{} {
	for k, v := range m {
		fields[k] = v
	}
	return fields
}

// Client is an client logging middleware.
func Client(opts ...Option) middleware.Middleware {
	cfg := Options{
		timeFormat:    defaultFormat,          // 默认时间格式
		logger:        logger.DefaultLogger,   // 默认日志
		slowThreshold: time.Millisecond * 300, // 默认慢日志时间
		handler: func(ctx context.Context, req any) map[string]string {
			return make(map[string]string)
		},
	}
	for _, o := range opts {
		o(&cfg)
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if cfg.disabled {
				return handler(ctx, req)
			}

			var (
				kind      string
				method    string
				callee    = "unknown"
				startTime = time.Now()
			)

			if peer, ok := selector.FromPeerContext(ctx); ok {
				callee = peer.Node.ServiceName()
			}

			if info, ok := transport.FromClientContext(ctx); ok {
				kind = info.Kind().String()
				method = info.Operation()
			}

			resp, err := handler(ctx, req)
			duration := time.Since(startTime)
			fields := map[string]interface{}{
				"start":     startTime.Format(cfg.timeFormat),
				"kind":      "client",
				"component": kind,
				"method":    method,
				"duration":  timex.Duration(duration),
				"callee":    callee,
			}
			if v := extractError(err); v != "" {
				fields["error"] = v
			}
			if se := errors.FromError(err); se != nil {
				fields["code"] = se.Code
				fields["reason"] = se.Reason
			}
			fields = mergeFields(fields, cfg.handler(ctx, req))

			log := cfg.logger.WithContext(ctx).WithFields(fields)

			// show log
			if duration > cfg.slowThreshold {
				log.Info(kind + " client slow")
			}
			if err != nil {
				log.Error(kind + " client")
			} else {
				log.Info(kind + " client")
			}

			return resp, err
		}
	}
}

// Server is an client logging middleware.
func Server(opts ...Option) middleware.Middleware {
	cfg := Options{
		timeFormat:    defaultFormat,          // 默认时间格式
		slowThreshold: time.Millisecond * 300, // 默认慢日志时间
		logger:        logger.DefaultLogger,   // 默认日志
		handler: func(ctx context.Context, req any) map[string]string {
			return make(map[string]string)
		},
	}
	for _, o := range opts {
		o(&cfg)
	}

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
			if cfg.disabled {
				return handler(ctx, req)
			}

			var (
				kind      string
				method    string
				caller    = "unknown"
				startTime = time.Now()
			)

			if info, ok := transport.FromServerContext(ctx); ok {
				kind = info.Kind().String()
				method = info.Operation()
			}

			resp, err := handler(ctx, req)
			duration := time.Since(startTime)
			fields := map[string]interface{}{
				"start":     startTime.Format(cfg.timeFormat),
				"kind":      "server",
				"component": kind,
				"method":    method,
				"duration":  timex.Duration(duration),
				"caller":    caller,
			}
			if se := errors.FromError(err); se != nil {
				fields["code"] = se.Code
				fields["reason"] = se.Reason
			}
			if v := extractError(err); v != "" {
				fields["error"] = extractError(err)
			}
			fields = mergeFields(fields, cfg.handler(ctx, req))

			log := cfg.logger.WithContext(ctx).WithFields(fields)
			// show log
			if duration > cfg.slowThreshold {
				log.Info(kind + " server slow")
			}
			if err != nil {
				log.Error(kind + " server")
			} else {
				log.Info(kind + " server")
			}

			return resp, err
		}
	}
}

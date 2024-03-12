package logging

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/selector"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/nextmicro/gokit/timex"
	"github.com/nextmicro/logger"
	config "github.com/nextmicro/next/api/config/v1"
	v1 "github.com/nextmicro/next/api/middleware/logging/v1"
	chain "github.com/nextmicro/next/middleware"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	defaultFormat = "2006-01-02T15:04:05.999Z0700"
)

const namespace = "logging"

func init() {
	chain.Register("client."+namespace, injectionClient)
	chain.Register("server."+namespace, injectionServer)
}

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

// mergeFields merges the fields
func mergeFields(fields map[string]interface{}, m map[string]string) map[string]interface{} {
	for k, v := range m {
		fields[k] = v
	}
	return fields
}

func injectionClient(c *config.Middleware) (middleware.Middleware, error) {
	v := durationpb.New(time.Millisecond * 300)
	options := &v1.Logging{
		TimeFormat:    defaultFormat,
		SlowThreshold: v,
	}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, options, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}

	opts := make([]Option, 0)
	if options.Disabled {
		opts = append(opts, WithDisabled(options.Disabled))
	}
	if options.TimeFormat != "" {
		opts = append(opts, WithTimeFormat(options.TimeFormat))
	}
	if options.SlowThreshold != nil && options.SlowThreshold.AsDuration() > 0 {
		opts = append(opts, WithSlowThreshold(options.SlowThreshold.AsDuration()))
	}

	return Client(opts...), nil
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
				kind        string
				route       string
				callee      = "unknown"
				startTime   = time.Now()
				nodeAddress = ""
			)

			if info, ok := transport.FromClientContext(ctx); ok {
				kind = info.Kind().String()
				route = info.Operation()
			}

			resp, err := handler(ctx, req)
			duration := time.Since(startTime)

			if peer, ok := selector.FromPeerContext(ctx); ok && peer.Node != nil {
				callee = peer.Node.ServiceName()
				nodeAddress = peer.Node.Address()
			}

			fields := map[string]interface{}{
				"start":     startTime.Format(cfg.timeFormat),
				"kind":      "client",
				"component": kind,
				"route":     route,
				"duration":  timex.Duration(duration),
				"callee":    callee,
			}
			if nodeAddress != "" {
				fields["callee.address"] = nodeAddress
			}

			if v := extractError(err); v != "" {
				fields["error"] = v
			}
			if se := errors.FromError(err); se != nil {
				fields["code"] = se.Code
				fields["reason"] = se.Reason
			}

			if cfg.handler != nil {
				fields = mergeFields(fields, cfg.handler(ctx, req))
			}

			_log := logger.WithContext(ctx).WithFields(fields)

			// show log
			if cfg.slowThreshold > 0 && duration > cfg.slowThreshold && err != nil {
				_log.Error(kind + " server slow")
			} else if cfg.slowThreshold > 0 && duration > cfg.slowThreshold {
				_log.Info(kind + " server slow")
			} else if err != nil {
				_log.Error(kind + " server")
			} else {
				_log.Info(kind + " server")
			}

			return resp, err
		}
	}
}

func injectionServer(c *config.Middleware) (middleware.Middleware, error) {
	v := durationpb.New(time.Millisecond * 300)
	options := &v1.Logging{
		TimeFormat:    defaultFormat,
		SlowThreshold: v,
	}

	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, options, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}

	opts := make([]Option, 0)
	if options.Disabled {
		opts = append(opts, WithDisabled(options.Disabled))
	}
	if options.TimeFormat != "" {
		opts = append(opts, WithTimeFormat(options.TimeFormat))
	}
	if options.SlowThreshold != nil && options.SlowThreshold.AsDuration() > 0 {
		opts = append(opts, WithSlowThreshold(options.SlowThreshold.AsDuration()))
	}

	return Server(opts...), nil
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
				route     string
				caller    = "unknown"
				startTime = time.Now()
			)

			if info, ok := transport.FromServerContext(ctx); ok {
				kind = info.Kind().String()
				route = info.Operation()
			}
			if md, ok := metadata.FromServerContext(ctx); ok {
				if v := md.Get("x-md-local-caller"); v != "" {
					caller = v
				}
			}

			resp, err := handler(ctx, req)
			duration := time.Since(startTime)
			fields := map[string]interface{}{
				"start":     startTime.Format(cfg.timeFormat),
				"kind":      "server",
				"component": kind,
				"route":     route,
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

			if cfg.handler != nil {
				fields = mergeFields(fields, cfg.handler(ctx, req))
			}

			_log := logger.WithContext(ctx).WithFields(fields)
			// show log
			if cfg.slowThreshold > 0 && duration > cfg.slowThreshold && err != nil {
				_log.Error(kind + " server slow")
			} else if cfg.slowThreshold > 0 && duration > cfg.slowThreshold {
				_log.Info(kind + " server slow")
			} else if err != nil {
				_log.Error(kind + " server")
			} else {
				_log.Info(kind + " server")
			}

			return resp, err
		}
	}
}

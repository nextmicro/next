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
	chain.Register("client."+namespace, Client)
	chain.Register("server."+namespace, Server)
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

// Client is an client logging middleware.
func Client(c *config.Middleware) (middleware.Middleware, error) {
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

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
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
				"start":     startTime.Format(options.TimeFormat),
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
			log := logger.WithContext(ctx).WithFields(fields)

			// show log
			if duration > options.GetSlowThreshold().AsDuration() {
				log.Info(kind + " client slow")
			} else if err != nil {
				log.Error(kind + " client")
			} else {
				log.Info(kind + " client")
			}

			return resp, err
		}
	}, nil
}

// Server is an client logging middleware.
func Server(c *config.Middleware) (middleware.Middleware, error) {
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

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (interface{}, error) {
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
				"start":     startTime.Format(options.GetTimeFormat()),
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

			log := logger.WithContext(ctx).WithFields(fields)
			// show log
			if duration > options.GetSlowThreshold().AsDuration() {
				log.Info(kind + " server slow")
			} else if err != nil {
				log.Error(kind + " server")
			} else {
				log.Info(kind + " server")
			}

			return resp, err
		}
	}, nil
}

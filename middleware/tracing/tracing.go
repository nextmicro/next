package tracing

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/nextmicro/gokit/trace/httpconv"
	configv1 "github.com/nextmicro/next/api/config/v1"
	v1 "github.com/nextmicro/next/api/middleware/tracing/v1"
	chain "github.com/nextmicro/next/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	tracerName = "github.com/nextmicro/next/middleware/tracing"
)

func init() {
	chain.Register("client.tracing", Client)
	chain.Register("server.tracing", Server)
}

// Client returns a new client middleware for OpenTelemetry.
func Client(c *configv1.Middleware) (middleware.Middleware, error) {
	opt := options{
		Tracing: &v1.Tracing{},
	}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, opt, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}

	cfg := options{
		tracerProvider: otel.GetTracerProvider(),
		propagators:    propagation.NewCompositeTextMapPropagator(Metadata{}, propagation.Baggage{}, propagation.TraceContext{}),
	}

	tracer := cfg.tracerProvider.Tracer(tracerName)

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromClientContext(ctx); ok {
				var span trace.Span

				var spanName = tr.Operation()
				switch tr.Kind() {
				case transport.KindHTTP:
					spanName = fmt.Sprintf("HTTP Client %s", tr.Operation())
				case transport.KindGRPC:
					spanName = fmt.Sprintf("GRPC Client %s", tr.Operation())
				}

				ctx, span = tracer.Start(ctx, spanName, oteltrace.WithSpanKind(oteltrace.SpanKindClient))
				defer span.End()

				cfg.propagators.Inject(ctx, tr.RequestHeader())

				setClientSpan(ctx, span, req)
				reply, err = handler(ctx, req)
				se := errors.FromError(err)
				switch tr.Kind() {
				case transport.KindHTTP:
					statusCode := http.StatusOK
					if se != nil {
						statusCode = int(se.GetCode())
					}
					attrs := httpconv.HTTPAttributesFromHTTPStatusCode(statusCode)
					spanStatus, spanMessage := httpconv.SpanStatusFromHTTPStatusCodeAndSpanKind(statusCode, oteltrace.SpanKindServer)
					span.SetAttributes(attrs...)
					span.SetStatus(spanStatus, spanMessage)
				case transport.KindGRPC:
					if err != nil {
						if se != nil {
							span.SetAttributes(semconv.RPCGRPCStatusCodeKey.Int64(int64(se.GRPCStatus().Code())))
						}
						span.SetStatus(codes.Error, err.Error())
					} else {
						span.SetStatus(codes.Ok, codes.Ok.String())
						span.SetAttributes(semconv.RPCGRPCStatusCodeOk)
					}
				}
			} else {
				reply, err = handler(ctx, req)
			}
			return
		}
	}, nil
}

// Server returns a new server middleware for OpenTelemetry.
func Server(c *configv1.Middleware) (middleware.Middleware, error) {
	opt := options{
		Tracing: &v1.Tracing{},
	}
	if c.Options != nil {
		if err := anypb.UnmarshalTo(c.Options, opt, proto.UnmarshalOptions{Merge: true}); err != nil {
			return nil, err
		}
	}

	cfg := options{
		tracerProvider: otel.GetTracerProvider(),
		propagators:    propagation.NewCompositeTextMapPropagator(Metadata{}, propagation.Baggage{}, propagation.TraceContext{}),
	}

	tracer := cfg.tracerProvider.Tracer(
		tracerName,
	)

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				header := tr.RequestHeader()
				ctx = cfg.propagators.Extract(ctx, header)
				bags := baggage.FromContext(ctx)
				spanCtx := oteltrace.SpanContextFromContext(ctx)
				ctx = baggage.ContextWithBaggage(ctx, bags)

				var spanName = tr.Operation()
				switch tr.Kind() {
				case transport.KindHTTP:
					spanName = fmt.Sprintf("HTTP Server %s", tr.Operation())
				case transport.KindGRPC:
					spanName = fmt.Sprintf("GRPC Server %s", tr.Operation())
				}

				var span trace.Span
				ctx, span = tracer.Start(oteltrace.ContextWithRemoteSpanContext(ctx, spanCtx), spanName, oteltrace.WithSpanKind(oteltrace.SpanKindServer))
				defer span.End()

				setServerSpan(ctx, span, req)

				reply, err = handler(ctx, req)
				se := errors.FromError(err)
				switch tr.Kind() {
				case transport.KindHTTP:
					statusCode := http.StatusOK
					if se != nil {
						statusCode = int(se.GetCode())
					}
					attrs := httpconv.HTTPAttributesFromHTTPStatusCode(statusCode)
					spanStatus, spanMessage := httpconv.SpanStatusFromHTTPStatusCodeAndSpanKind(statusCode, oteltrace.SpanKindServer)
					span.SetAttributes(attrs...)
					span.SetStatus(spanStatus, spanMessage)
				case transport.KindGRPC:
					if err != nil {
						span.RecordError(err)
						if se != nil {
							span.SetAttributes(semconv.RPCGRPCStatusCodeKey.Int64(int64(se.GRPCStatus().Code())))
						}
						span.SetStatus(codes.Error, err.Error())
					} else {
						span.SetStatus(codes.Ok, codes.Ok.String())
						span.SetAttributes(semconv.RPCGRPCStatusCodeOk)
					}
				}

				tr.ReplyHeader().Set("x-span-id", span.SpanContext().SpanID().String())
				tr.ReplyHeader().Set("x-trace-id", span.SpanContext().TraceID().String())

			} else {
				reply, err = handler(ctx, req)
			}

			return
		}
	}, nil
}

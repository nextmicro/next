package tracing

import (
	"context"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-volo/logger"
	configv1 "github.com/nextmicro/next/api/config/v1"
	chain "github.com/nextmicro/next/middleware"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
	"go.opentelemetry.io/otel/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "github.com/nextmicro/next/middleware/tracing"
)

func init() {
	chain.Register("tracing.client", Client)
}

// TraceID returns a traceid valuer.
func TraceID() logger.Valuer {
	return func(ctx context.Context) interface{} {
		if span := trace.SpanContextFromContext(ctx); span.HasTraceID() {
			return span.TraceID().String()
		}
		return ""
	}
}

// SpanID returns a spanid valuer.
func SpanID() logger.Valuer {
	return func(ctx context.Context) interface{} {
		if span := trace.SpanContextFromContext(ctx); span.HasSpanID() {
			return span.SpanID().String()
		}
		return ""
	}
}

// Client returns a new client middleware for OpenTelemetry.
func Client(c *configv1.Middleware) (middleware.Middleware, error) {
	cfg := options{
		tracerProvider: otel.GetTracerProvider(),
		propagators:    propagation.NewCompositeTextMapPropagator(Metadata{}, propagation.Baggage{}, propagation.TraceContext{}),
	}

	tracer := cfg.tracerProvider.Tracer(tracerName)

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromClientContext(ctx); ok {
				var span trace.Span
				ctx, span = tracer.Start(ctx, tr.Operation(), oteltrace.WithSpanKind(oteltrace.SpanKindClient))
				defer span.End()

				cfg.propagators.Inject(ctx, tr.RequestHeader())

				setClientSpan(ctx, span, req)
				reply, err = handler(ctx, req)
				se := errors.FromError(err)
				switch tr.Kind() {
				case transport.KindHTTP:
					if err != nil {
						span.RecordError(err)
						if se != nil {
							span.SetAttributes(semconv.HTTPStatusCodeKey.Int64(int64(se.Code)))
						}
						span.SetStatus(codes.Error, err.Error())
					} else {
						span.SetStatus(codes.Ok, "OK")
					}
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

				var span trace.Span
				ctx, span = tracer.Start(oteltrace.ContextWithRemoteSpanContext(ctx, spanCtx), tr.Operation(), oteltrace.WithSpanKind(oteltrace.SpanKindServer))
				defer span.End()

				setServerSpan(ctx, span, req)

				reply, err = handler(ctx, req)
				se := errors.FromError(err)
				switch tr.Kind() {
				case transport.KindHTTP:
					if err != nil {
						span.RecordError(err)
						if se != nil {
							span.SetAttributes(semconv.HTTPStatusCodeKey.Int64(int64(se.Code)))
						}
						span.SetStatus(codes.Error, err.Error())
					} else {
						span.SetStatus(codes.Ok, "OK")
					}

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
			} else {
				reply, err = handler(ctx, req)
			}

			return
		}
	}, nil
}

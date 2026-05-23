package tracing

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kratos/kratos/v2/errors"
	ktracing "github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/nextmicro/gokit/trace/httpconv"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/nextmicro/next/middleware/tracing"

// Client returns client tracing middleware.
// Span lifecycle uses kratos Tracer; enrichment/end semantics preserve next conventions.
func Client(opts ...Option) middleware.Middleware {
	cfg := defaultOptions()
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	if cfg.disabled {
		return func(h middleware.Handler) middleware.Handler { return h }
	}

	tracer := ktracing.NewTracer(
		trace.SpanKindClient,
		ktracing.WithPropagator(cfg.propagators),
		ktracing.WithTracerProvider(cfg.tracerProvider),
		ktracing.WithTracerName(tracerName),
	)

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			tr, ok := transport.FromClientContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			ctx, span := tracer.Start(ctx, spanName(tr, false), tr.RequestHeader())
			defer span.End()

			setClientSpan(ctx, span, req)
			reply, err = handler(ctx, req)
			recordClientSpanStatus(span, tr, err)
			return reply, err
		}
	}
}

// Server returns server tracing middleware.
func Server(opts ...Option) middleware.Middleware {
	cfg := defaultOptions()
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	if cfg.disabled {
		return func(h middleware.Handler) middleware.Handler { return h }
	}

	tracer := ktracing.NewTracer(
		trace.SpanKindServer,
		ktracing.WithPropagator(cfg.propagators),
		ktracing.WithTracerProvider(cfg.tracerProvider),
		ktracing.WithTracerName(tracerName),
	)

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			tr, ok := transport.FromServerContext(ctx)
			if !ok {
				return handler(ctx, req)
			}

			ctx, span := tracer.Start(ctx, spanName(tr, true), tr.RequestHeader())
			defer span.End()

			setServerSpan(ctx, span, req)
			reply, err = handler(ctx, req)
			recordServerSpanStatus(span, tr, err)
			tr.ReplyHeader().Set("x-span-id", span.SpanContext().SpanID().String())
			tr.ReplyHeader().Set("x-trace-id", span.SpanContext().TraceID().String())
			return reply, err
		}
	}
}

func spanName(tr transport.Transporter, server bool) string {
	op := tr.Operation()
	switch tr.Kind() {
	case transport.KindHTTP:
		if server {
			return fmt.Sprintf("HTTP Server %s", op)
		}
		return fmt.Sprintf("HTTP Client %s", op)
	case transport.KindGRPC:
		if server {
			return fmt.Sprintf("GRPC Server %s", op)
		}
		return fmt.Sprintf("GRPC Client %s", op)
	default:
		return op
	}
}

func recordClientSpanStatus(span trace.Span, tr transport.Transporter, err error) {
	se := errors.FromError(err)
	switch tr.Kind() {
	case transport.KindHTTP:
		statusCode := http.StatusOK
		if se != nil {
			statusCode = int(se.GetCode())
		}
		attrs := httpconv.HTTPAttributesFromHTTPStatusCode(statusCode)
		st, msg := httpconv.SpanStatusFromHTTPStatusCodeAndSpanKind(statusCode, trace.SpanKindServer)
		span.SetAttributes(attrs...)
		span.SetStatus(st, msg)
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
}

func recordServerSpanStatus(span trace.Span, tr transport.Transporter, err error) {
	se := errors.FromError(err)
	switch tr.Kind() {
	case transport.KindHTTP:
		statusCode := http.StatusOK
		if se != nil {
			statusCode = int(se.GetCode())
		}
		attrs := httpconv.HTTPAttributesFromHTTPStatusCode(statusCode)
		st, msg := httpconv.SpanStatusFromHTTPStatusCodeAndSpanKind(statusCode, trace.SpanKindServer)
		span.SetAttributes(attrs...)
		span.SetStatus(st, msg)
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
}

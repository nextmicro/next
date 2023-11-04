package tracing

import (
	"context"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/nextmicro/gokit/trace/httpconv"
	thttp "github.com/nextmicro/next/transport/http"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/proto"
)

func setClientSpan(ctx context.Context, span trace.Span, request any) {
	attrs := make([]attribute.KeyValue, 0, 3)
	var remote string
	var operation string
	var rpcKind string
	tr, ok := transport.FromClientContext(ctx)
	if ok {
		operation = tr.Operation()
		rpcKind = tr.Kind().String()
		switch tr.Kind() {
		case transport.KindHTTP:
			if ht, ok := tr.(thttp.Transporter); ok {
				attrs = httpconv.ClientRequest(ht.Request())
				remote = ht.Request().Host
			}
		case transport.KindGRPC:
			remote, _ = parseTarget(tr.Endpoint())
			_, mAttrs := parseFullMethod(operation)
			attrs = append(attrs, mAttrs...)
		}
	}

	attrs = append(attrs, semconv.RPCSystemKey.String(rpcKind))
	if remote != "" {
		attrs = append(attrs, peerAttr(remote)...)
	}
	if p, ok := request.(proto.Message); ok {
		attrs = append(attrs, attribute.Key("send_msg.size").Int(proto.Size(p)))
	}

	span.SetAttributes(attrs...)
}

func setServerSpan(ctx context.Context, span trace.Span, request any) {
	attrs := make([]attribute.KeyValue, 0)
	var (
		remote      string
		rpcKind     string
		operation   string
		serviceName string
	)

	if md, ok := metadata.FromServerContext(ctx); ok {
		serviceName = md.Get(serviceHeader)
		attrs = append(attrs, semconv.PeerServiceKey.String(serviceName))
	}

	tr, ok := transport.FromServerContext(ctx)
	if ok {
		operation = tr.Operation()
		rpcKind = tr.Kind().String()
		switch tr.Kind() {
		case transport.KindHTTP:
			if ht, ok := tr.(thttp.Transporter); ok {
				attrs = append(attrs, httpconv.ServerRequest("", ht.Request())...)
			}
		case transport.KindGRPC:
			if p, ok := peer.FromContext(ctx); ok {
				remote = p.Addr.String()
			}
			_, mAttrs := parseFullMethod(operation)
			attrs = append(attrs, mAttrs...)
		}
	}
	attrs = append(attrs, semconv.RPCSystemKey.String(rpcKind))
	attrs = append(attrs, peerAttr(remote)...)
	span.SetAttributes(attrs...)
}

// peerAttr returns attributes about the peer address.
func peerAttr(addr string) []attribute.KeyValue {
	host, p, err := net.SplitHostPort(addr)
	if err != nil {
		return []attribute.KeyValue(nil)
	}

	if host == "" {
		host = "127.0.0.1"
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return []attribute.KeyValue(nil)
	}

	var attr []attribute.KeyValue
	if ip := net.ParseIP(host); ip != nil {
		attr = []attribute.KeyValue{
			semconv.NetSockPeerAddrKey.String(host),
			semconv.NetSockPeerPortKey.Int(port),
		}
	} else {
		attr = []attribute.KeyValue{
			semconv.NetPeerNameKey.String(host),
			semconv.NetPeerPortKey.Int(port),
		}
	}

	return attr
}

func parseTarget(endpoint string) (address string, err error) {
	var u *url.URL
	u, err = url.Parse(endpoint)
	if err != nil {
		if u, err = url.Parse("http://" + endpoint); err != nil {
			return "", err
		}
		return u.Host, nil
	}
	if len(u.Path) > 1 {
		return u.Path[1:], nil
	}
	return endpoint, nil
}

// parseFullMethod returns a span name following the OpenTelemetry semantic
// conventions as well as all applicable span attribute.KeyValue attributes based
// on a gRPC's FullMethod.
func parseFullMethod(fullMethod string) (string, []attribute.KeyValue) {
	name := strings.TrimLeft(fullMethod, "/")
	parts := strings.SplitN(name, "/", 2)
	if len(parts) != 2 { //nolint:gomnd
		// Invalid format, does not follow `/package.service/method`.
		return name, []attribute.KeyValue{attribute.Key("rpc.operation").String(fullMethod)}
	}

	var attrs []attribute.KeyValue
	if service := parts[0]; service != "" {
		attrs = append(attrs, semconv.RPCServiceKey.String(service))
	}
	if method := parts[1]; method != "" {
		attrs = append(attrs, semconv.RPCMethodKey.String(method))
	}
	return name, attrs
}

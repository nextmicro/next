package logging_test

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/nextmicro/next/middleware/logging"
	"github.com/stretchr/testify/assert"
)

// MockHandler 模拟的请求处理函数
func MockHandler(ctx context.Context, req interface{}) (interface{}, error) {
	return nil, nil
}

type Transport struct {
	kind      transport.Kind
	endpoint  string
	operation string
	reqHeader transport.Header
}

func (tr *Transport) Kind() transport.Kind {
	return tr.kind
}

func (tr *Transport) Endpoint() string {
	return tr.endpoint
}

func (tr *Transport) Operation() string {
	return tr.operation
}

func (tr *Transport) RequestHeader() transport.Header {
	return tr.reqHeader
}

func (tr *Transport) ReplyHeader() transport.Header {
	return nil
}

func TestClient(t *testing.T) {
	// 构造请求处理函数
	handler := logging.Client(
		logging.WithIgnoredRoutes([]string{"/users/{name}"}), // 忽略 /users/{name} 路由的日志输出
		logging.WithMetadata([]logging.Metadata{
			{
				Key: "x-md-global-extra",
			},
			{
				Key:    "x-md-global-uid",
				Rename: "uid",
			},
			{
				Key: "trace_id",
			},
		}),
	)(MockHandler)

	// 执行请求处理函数

	ctx := transport.NewClientContext(context.Background(), &Transport{operation: "/users/{name}"})
	resp, err := handler(ctx, nil)

	// 断言期望的日志输出和返回值
	assert.NoError(t, err)
	assert.Nil(t, resp)

	// 执行请求处理函数
	ctx = metadata.AppendToClientContext(context.Background(),
		"x-md-global-extra", "2233",
		"x-md-global-uid", "1234",
		"trace_id", "1234567890",
	)
	ctx = transport.NewClientContext(ctx, &Transport{operation: "/users/me"})
	resp, err = handler(ctx, nil)

	// 断言期望的日志输出和返回值
	assert.NoError(t, err)
	assert.Nil(t, resp)
}

func TestServer(t *testing.T) {
	// 构造请求处理函数
	handler := logging.Server(
		logging.WithIgnoredRoutes([]string{"/users/{name}"}), // 忽略 /users/{name} 路由的日志输出
		logging.WithMetadata([]logging.Metadata{
			{
				Key: "x-md-global-extra",
			},
			{
				Key:    "x-md-global-uid",
				Rename: "uid",
			},
			{
				Key: "trace_id",
			},
		}),
	)(MockHandler)

	// 执行请求处理函数

	ctx := transport.NewServerContext(context.Background(), &Transport{operation: "/users/{name}"})
	resp, err := handler(ctx, nil)

	// 断言期望的日志输出和返回值
	assert.NoError(t, err)
	assert.Nil(t, resp)

	// 执行请求处理函数
	serverMD := metadata.New()
	serverMD.Set("x-md-global-extra", "!23")
	serverMD.Set("x-md-global-uid", "1000")
	serverMD.Set("trace_id", "1234567890")
	ctx = metadata.NewServerContext(context.Background(), serverMD)
	ctx = transport.NewServerContext(ctx, &Transport{operation: "/users/me"})
	resp, err = handler(ctx, nil)

	// 断言期望的日志输出和返回值
	assert.NoError(t, err)
	assert.Nil(t, resp)
}

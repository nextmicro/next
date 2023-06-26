package grpc

import (
	"context"

	client "github.com/go-kratos/kratos/v2/transport/grpc"
	conf "github.com/nextmicro/next/config"
	"github.com/nextmicro/next/middleware"
	"google.golang.org/grpc"
)

type Client struct {
	*grpc.ClientConn
}

func NewClient(ctx context.Context, opts ...client.ClientOption) (*Client, error) {
	ms, err := middleware.BuildMiddleware(conf.ApplicationConfig().GetMiddlewares())
	if err != nil {
		return nil, err
	}

	grpcOpts := make([]client.ClientOption, 0, len(opts)+1)
	grpcOpts = append(grpcOpts, client.WithMiddleware(ms...))
	grpcOpts = append(grpcOpts, opts...)
	conn, err := client.DialInsecure(ctx, grpcOpts...)
	if err != nil {
		return nil, err
	}
	return &Client{
		ClientConn: conn,
	}, nil
}

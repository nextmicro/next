package http

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport/http"
	conf "github.com/nextmicro/next/config"
	"github.com/nextmicro/next/middleware"
)

type Client struct {
	*http.Client
}

// NewClient returns an HTTP client.
func NewClient(ctx context.Context, opts ...http.ClientOption) (*Client, error) {
	ms, err := middleware.BuildMiddleware(conf.ApplicationConfig().GetMiddlewares())
	if err != nil {
		return nil, err
	}

	httpOpts := make([]http.ClientOption, 0, len(opts)+1)
	httpOpts = append(httpOpts, http.WithMiddleware(ms...))
	httpOpts = append(httpOpts, opts...)
	client, err := http.NewClient(ctx, httpOpts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		Client: client,
	}, nil
}

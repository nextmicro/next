package metrics

import (
	"github.com/go-kratos/kratos/v2/middleware"
	configv1 "github.com/nextmicro/next/api/config/v1"
	chain "github.com/nextmicro/next/middleware"
)

const namespace = "metrics"

func init() {
	chain.Register("client."+namespace, factoryClient)
	chain.Register("server."+namespace, factoryServer)
}

func factoryClient(c *configv1.Middleware) (middleware.Middleware, error) {
	opts, err := optionsFromConfig(c)
	if err != nil {
		return nil, err
	}
	opts = append(defaultClientOptions(), opts...)
	return Client(opts...), nil
}

func factoryServer(c *configv1.Middleware) (middleware.Middleware, error) {
	opts, err := optionsFromConfig(c)
	if err != nil {
		return nil, err
	}
	opts = append(defaultServerOptions(), opts...)
	return Server(opts...), nil
}

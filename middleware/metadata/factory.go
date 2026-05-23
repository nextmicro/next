package metadata

import (
	"github.com/go-kratos/kratos/v2/middleware"
	configv1 "github.com/nextmicro/next/api/config/v1"
	chain "github.com/nextmicro/next/middleware"
)

const namespace = "metadata"

func init() {
	chain.Register("client."+namespace, factoryClient)
	chain.Register("server."+namespace, factoryServer)
}

func factoryServer(c *configv1.Middleware) (middleware.Middleware, error) {
	opts, err := optionsFromConfigServer(c)
	if err != nil {
		return nil, err
	}
	return Server(opts...), nil
}

func factoryClient(c *configv1.Middleware) (middleware.Middleware, error) {
	opts, err := optionsFromConfigClient(c)
	if err != nil {
		return nil, err
	}
	return Client(opts...), nil
}

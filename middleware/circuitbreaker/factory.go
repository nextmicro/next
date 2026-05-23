package circuitbreaker

import (
	"github.com/go-kratos/kratos/v2/middleware"
	configv1 "github.com/nextmicro/next/api/config/v1"
	chain "github.com/nextmicro/next/middleware"
)

func init() {
	chain.Register("client.circuitbreaker", factory)
}

func factory(c *configv1.Middleware) (middleware.Middleware, error) {
	opts, err := optionsFromConfig(c)
	if err != nil {
		return nil, err
	}
	return Client(opts...), nil
}

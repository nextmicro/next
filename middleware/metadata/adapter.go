package metadata

import (
	kmetadata "github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware"
)

// Option is metadata option (kratos adapter).
type Option = kmetadata.Option

// WithConstants with constant metadata key value.
var WithConstants = kmetadata.WithConstants

// WithPropagatedPrefix with propagated key prefix.
var WithPropagatedPrefix = kmetadata.WithPropagatedPrefix

// Server is middleware server-side metadata.
func Server(opts ...Option) middleware.Middleware {
	return kmetadata.Server(opts...)
}

// Client is middleware client-side metadata.
func Client(opts ...Option) middleware.Middleware {
	return kmetadata.Client(opts...)
}

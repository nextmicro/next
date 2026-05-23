package recovery

import (
	krecovery "github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware"
)

// Option is recovery option (kratos adapter).
type Option = krecovery.Option

// ErrPanicRecover is panic recover error (next convention).
var ErrPanicRecover = krecovery.ErrUnknownRequest

// WithHandler with recovery handler.
var WithHandler = krecovery.WithHandler

// Recovery is a server middleware that recovers from any panics.
func Recovery(opts ...Option) middleware.Middleware {
	return krecovery.Recovery(opts...)
}

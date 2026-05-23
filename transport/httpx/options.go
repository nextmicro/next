package httpx

import (
	"net/http"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

// RouterOptions configures httpx router codec and error handling behavior.
type RouterOptions struct {
	DecVars  DecodeRequestFunc
	DecQuery DecodeRequestFunc
	DecBody  DecodeRequestFunc
	Enc      EncodeResponseFunc
	EncErr   EncodeErrorFunc
}

// RouterOption configures RouterOptions.
type RouterOption func(*RouterOptions)

// WithRequestVarsDecoder sets the vars decoder.
func WithRequestVarsDecoder(dec DecodeRequestFunc) RouterOption {
	return func(o *RouterOptions) { o.DecVars = dec }
}

// WithRequestQueryDecoder sets the query decoder.
func WithRequestQueryDecoder(dec DecodeRequestFunc) RouterOption {
	return func(o *RouterOptions) { o.DecQuery = dec }
}

// WithRequestDecoder sets the body decoder.
func WithRequestDecoder(dec DecodeRequestFunc) RouterOption {
	return func(o *RouterOptions) { o.DecBody = dec }
}

// WithResponseEncoder sets the response encoder.
func WithResponseEncoder(enc EncodeResponseFunc) RouterOption {
	return func(o *RouterOptions) { o.Enc = enc }
}

// WithErrorEncoder sets the error encoder.
func WithErrorEncoder(enc EncodeErrorFunc) RouterOption {
	return func(o *RouterOptions) { o.EncErr = enc }
}

func defaultRouterOptions() RouterOptions {
	return RouterOptions{
		DecVars:  DefaultRequestVars,
		DecQuery: DefaultRequestQuery,
		DecBody:  DefaultRequestDecoder,
		Enc:      DefaultResponseEncoder,
		EncErr:   DefaultErrorEncoder,
	}
}

func applyRouterOptions(opts []RouterOption) RouterOptions {
	cfg := defaultRouterOptions()
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

// ServerOptionsFromRouter returns kratos server options that mirror httpx codec defaults.
func ServerOptionsFromRouter(opts ...RouterOption) []khttp.ServerOption {
	cfg := applyRouterOptions(opts)
	return []khttp.ServerOption{
		khttp.RequestVarsDecoder(cfg.DecVars),
		khttp.RequestQueryDecoder(cfg.DecQuery),
		khttp.RequestDecoder(cfg.DecBody),
		khttp.ResponseEncoder(cfg.Enc),
		khttp.ErrorEncoder(func(w http.ResponseWriter, r *http.Request, err error) {
			res := NewResponse(w)
			c := &wrapper{
				router: &router{opts: cfg},
				req:    r,
				res:    res,
			}
			cfg.EncErr(c, err)
		}),
	}
}

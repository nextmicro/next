package httpx

import (
	"net/http"
	"sync"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
)

type router struct {
	srv           *khttp.Server
	krouter       *khttp.Router
	preMiddleware []MiddlewareFunc
	pool          sync.Pool
	opts          RouterOptions
}

// NewRouter builds a next Router DSL on top of a kratos HTTP server.
func NewRouter(srv *khttp.Server, prefix string, m ...MiddlewareFunc) Router {
	return newRouter(srv, prefix, defaultRouterOptions(), m...)
}

func newRouter(srv *khttp.Server, prefix string, opts RouterOptions, m ...MiddlewareFunc) *router {
	r := &router{
		srv:           srv,
		krouter:       srv.Route(prefix),
		preMiddleware: append([]MiddlewareFunc(nil), m...),
		opts:          opts,
	}
	r.pool.New = func() any { return newContext(r) }
	return r
}

func (r *router) Use(m ...MiddlewareFunc) Router {
	r.preMiddleware = append(r.preMiddleware, m...)
	return r
}

func (r *router) Group(prefix string, m ...MiddlewareFunc) Router {
	var merged []MiddlewareFunc
	merged = append(merged, r.preMiddleware...)
	merged = append(merged, m...)
	return &router{
		srv:           r.srv,
		krouter:       r.krouter.Group(prefix),
		preMiddleware: merged,
		pool:          r.pool,
		opts:          r.opts,
	}
}

func (r *router) GET(path string, h HandlerFunc, m ...MiddlewareFunc) {
	r.Handle(http.MethodGet, path, h, m...)
}

func (r *router) HEAD(path string, h HandlerFunc, m ...MiddlewareFunc) {
	r.Handle(http.MethodHead, path, h, m...)
}

func (r *router) POST(path string, h HandlerFunc, m ...MiddlewareFunc) {
	r.Handle(http.MethodPost, path, h, m...)
}

func (r *router) PUT(path string, h HandlerFunc, m ...MiddlewareFunc) {
	r.Handle(http.MethodPut, path, h, m...)
}

func (r *router) PATCH(path string, h HandlerFunc, m ...MiddlewareFunc) {
	r.Handle(http.MethodPatch, path, h, m...)
}

func (r *router) DELETE(path string, h HandlerFunc, m ...MiddlewareFunc) {
	r.Handle(http.MethodDelete, path, h, m...)
}

func (r *router) CONNECT(path string, h HandlerFunc, m ...MiddlewareFunc) {
	r.Handle(http.MethodConnect, path, h, m...)
}

func (r *router) OPTIONS(path string, h HandlerFunc, m ...MiddlewareFunc) {
	r.Handle(http.MethodOptions, path, h, m...)
}

func (r *router) TRACE(path string, h HandlerFunc, m ...MiddlewareFunc) {
	r.Handle(http.MethodTrace, path, h, m...)
}

func (r *router) Handle(method, relativePath string, h HandlerFunc, middleware ...MiddlewareFunc) {
	chain := applyMiddleware(h, middleware...)
	chain = applyMiddleware(chain, r.preMiddleware...)

	kh := func(kctx khttp.Context) error {
		ctx := r.pool.Get().(Context)
		wc := ctx.(*wrapper)
		wc.router = r
		wc.Reset(NewResponse(kctx.Response()), kctx.Request())
		wc.kctx = kctx

		if err := chain(ctx); err != nil {
			r.opts.EncErr(ctx, err)
		}

		wc.Reset(nil, nil)
		r.pool.Put(ctx)
		return nil
	}

	r.krouter.Handle(method, relativePath, kh)
}

func applyMiddleware(h HandlerFunc, middleware ...MiddlewareFunc) HandlerFunc {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}

// WrapHandler wraps http.Handler into HandlerFunc.
func WrapHandler(h http.Handler) HandlerFunc {
	return func(c Context) error {
		h.ServeHTTP(c.Response(), c.Request())
		return nil
	}
}

// WrapMiddleware wraps func(http.Handler) http.Handler into MiddlewareFunc.
func WrapMiddleware(m func(http.Handler) http.Handler) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) (err error) {
			m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				c.SetRequest(r)
				c.SetResponse(NewResponse(w))
				err = next(c)
			})).ServeHTTP(c.Response(), c.Request())
			return
		}
	}
}

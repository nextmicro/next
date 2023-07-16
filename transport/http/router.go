package http

import (
	"net/http"
	"path"
	"sync"
)

// WalkRouteFunc is the type of the function called for each route visited by Walk.
type WalkRouteFunc func(RouteInfo) error

// RouteInfo is an HTTP route info.
type RouteInfo struct {
	Path   string
	Method string
}

type Handler interface {
	Handle(router Router)
}

type (
	// MiddlewareFunc defines a function to process middleware.
	MiddlewareFunc func(next HandlerFunc) HandlerFunc

	// HandlerFunc defines a function to serve HTTP requests.
	HandlerFunc func(Context) error
)

type Router interface {
	GET(path string, handler HandlerFunc, m ...MiddlewareFunc)
	POST(path string, handler HandlerFunc, m ...MiddlewareFunc)
	PUT(path string, handler HandlerFunc, m ...MiddlewareFunc)
	DELETE(path string, handler HandlerFunc, m ...MiddlewareFunc)
	HEAD(path string, handler HandlerFunc, m ...MiddlewareFunc)
	PATCH(path string, handler HandlerFunc, m ...MiddlewareFunc)
	CONNECT(path string, h HandlerFunc, m ...MiddlewareFunc)
	OPTIONS(path string, h HandlerFunc, m ...MiddlewareFunc)
	TRACE(path string, h HandlerFunc, m ...MiddlewareFunc)
	Group(prefix string, filters ...MiddlewareFunc) Router
}

// Router is an HTTP router.
type router struct {
	prefix        string
	pool          sync.Pool
	srv           *Server
	preMiddleware []MiddlewareFunc
}

func newRouter(prefix string, srv *Server, middleware ...MiddlewareFunc) *router {
	r := &router{
		prefix:        prefix,
		srv:           srv,
		preMiddleware: middleware,
	}
	r.pool.New = func() interface{} {
		return &wrapper{router: r}
	}
	return r
}

// Handle registers a new route with a matcher for the URL path and method.
func (r *router) Handle(method, relativePath string, h HandlerFunc, middleware ...MiddlewareFunc) {
	var next func(Context) error
	if r.preMiddleware == nil {
		next = applyMiddleware(h, middleware...)
	} else {
		next = applyMiddleware(h, middleware...)
		next = applyMiddleware(next, r.preMiddleware...)
	}

	handler := http.Handler(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		ctx := r.pool.Get().(Context)
		ctx.Reset(res, req)
		if err := next(ctx); err != nil {
			r.srv.ene(res, req, err)
		}
		ctx.Reset(nil, nil)
		r.pool.Put(ctx)
	}))

	r.srv.router.Handle(path.Join(r.prefix, relativePath), handler).Methods(method)
}

// Group returns a new router group.
func (r *router) Group(prefix string, middleware ...MiddlewareFunc) Router {
	var newMiddleware []MiddlewareFunc
	newMiddleware = append(newMiddleware, r.preMiddleware...)
	newMiddleware = append(newMiddleware, middleware...)
	return newRouter(path.Join(r.prefix, prefix), r.srv, newMiddleware...)
}

// GET registers a new GET route for a path with matching handler in the router.
func (r *router) GET(path string, h HandlerFunc, m ...MiddlewareFunc) {
	r.Handle(http.MethodGet, path, h, m...)
}

// HEAD registers a new HEAD route for a path with matching handler in the router.
func (r *router) HEAD(path string, h HandlerFunc, m ...MiddlewareFunc) {
	r.Handle(http.MethodHead, path, h, m...)
}

// POST registers a new POST route for a path with matching handler in the router.
func (r *router) POST(path string, h HandlerFunc, m ...MiddlewareFunc) {
	r.Handle(http.MethodPost, path, h, m...)
}

// PUT registers a new PUT route for a path with matching handler in the router.
func (r *router) PUT(path string, h HandlerFunc, m ...MiddlewareFunc) {
	r.Handle(http.MethodPut, path, h, m...)
}

// PATCH registers a new PATCH route for a path with matching handler in the router.
func (r *router) PATCH(path string, h HandlerFunc, m ...MiddlewareFunc) {
	r.Handle(http.MethodPatch, path, h, m...)
}

// DELETE registers a new DELETE route for a path with matching handler in the router.
func (r *router) DELETE(path string, h HandlerFunc, m ...MiddlewareFunc) {
	r.Handle(http.MethodDelete, path, h, m...)
}

// CONNECT registers a new CONNECT route for a path with matching handler in the router.
func (r *router) CONNECT(path string, h HandlerFunc, m ...MiddlewareFunc) {
	r.Handle(http.MethodConnect, path, h, m...)
}

// OPTIONS registers a new OPTIONS route for a path with matching handler in the router.
func (r *router) OPTIONS(path string, h HandlerFunc, m ...MiddlewareFunc) {
	r.Handle(http.MethodOptions, path, h, m...)
}

// TRACE registers a new TRACE route for a path with matching handler in the router.
func (r *router) TRACE(path string, h HandlerFunc, m ...MiddlewareFunc) {
	r.Handle(http.MethodTrace, path, h, m...)
}

func applyMiddleware(h HandlerFunc, middleware ...MiddlewareFunc) HandlerFunc {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}

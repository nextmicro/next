package httpx

import (
	"context"
	"io"
	"net/http"
	"net/url"

	"github.com/go-kratos/kratos/v2/middleware"
)

// WalkRouteFunc is the type of the function called for each route visited by Walk.
type WalkRouteFunc func(RouteInfo) error

// RouteInfo is an HTTP route info.
type RouteInfo struct {
	Path   string
	Method string
}

// Handler registers routes on a Router.
type Handler interface {
	Prefix() string
	Handle(router Router)
}

// BaseHandler is a no-op Handler useful for embedding.
type BaseHandler struct{}

func (*BaseHandler) Prefix() string { return "" }
func (*BaseHandler) Handle(Router)  {}

// MiddlewareFunc defines a function to process middleware.
type MiddlewareFunc func(next HandlerFunc) HandlerFunc

// HandlerFunc defines a function to serve HTTP requests.
type HandlerFunc func(Context) error

// Router is the next HTTP router DSL built on kratos *http.Server.
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
	Group(prefix string, m ...MiddlewareFunc) Router
	Use(m ...MiddlewareFunc) Router
}

// Context is an HTTP Context for handler functions.
type Context interface {
	context.Context
	Vars() url.Values
	Query() url.Values
	Form() url.Values
	Header() http.Header
	Request() *http.Request
	SetRequest(r *http.Request)
	Response() *Response
	SetResponse(r *Response)
	Middleware(middleware.Handler) middleware.Handler
	Bind(v interface{}) error
	BindVars(v interface{}) error
	BindQuery(v interface{}) error
	BindForm(v interface{}) error
	Returns(v interface{}, err error) error
	Result(code int, v interface{}) error
	JSON(code int, v interface{}) error
	XML(code int, v interface{}) error
	String(code int, text string) error
	Blob(code int, contentType string, data []byte) error
	Stream(code int, contentType string, rd io.Reader) error
	Reset(http.ResponseWriter, *http.Request)
}

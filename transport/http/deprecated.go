// Package http 提供 v1 兼容层，底层已迁移至 kratos 原生 transport 与 transport/httpx Router DSL。
//
// Deprecated: 新业务请使用 transport.NewHTTPServer + transport/httpx。
package http

import (
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/nextmicro/next/transport/httpx"
)

// Server wraps kratos HTTP server and preserves Route() with next MiddlewareFunc DSL.
//
// Deprecated: use transport.NewHTTPServer returning *khttp.Server and httpx.NewRouter.
type Server struct {
	*khttp.Server
}

// ServerOption configures the HTTP server.
//
// Deprecated: use khttp.ServerOption.
type ServerOption = khttp.ServerOption

// FilterFunc is an HTTP filter function.
//
// Deprecated: use khttp.FilterFunc.
type FilterFunc = khttp.FilterFunc

// FilterChain returns a chained FilterFunc.
//
// Deprecated: use khttp.FilterChain.
var FilterChain = khttp.FilterChain

// DecodeRequestFunc decodes HTTP requests.
//
// Deprecated: use khttp.DecodeRequestFunc.
type DecodeRequestFunc = khttp.DecodeRequestFunc

// EncodeResponseFunc encodes HTTP responses.
//
// Deprecated: use khttp.EncodeResponseFunc.
type EncodeResponseFunc = khttp.EncodeResponseFunc

// EncodeErrorFunc encodes HTTP errors.
//
// Deprecated: use khttp.EncodeErrorFunc.
type EncodeErrorFunc = khttp.EncodeErrorFunc

// WalkRouteFunc walks registered routes.
type WalkRouteFunc = httpx.WalkRouteFunc

// RouteInfo describes a registered route.
type RouteInfo = httpx.RouteInfo

// Context is the next HTTP handler context.
type Context = httpx.Context

// Router is the next HTTP router DSL.
type Router = httpx.Router

// HandlerFunc serves HTTP requests.
type HandlerFunc = httpx.HandlerFunc

// MiddlewareFunc wraps HandlerFunc middleware.
type MiddlewareFunc = httpx.MiddlewareFunc

// Handler registers routes on a Router.
type Handler = httpx.Handler

// BaseHandler is a no-op Handler useful for embedding.
type BaseHandler = httpx.BaseHandler

// Response wraps http.ResponseWriter.
type Response = httpx.Response

// CustomResponse is the next standard HTTP response envelope.
type CustomResponse = httpx.CustomResponse

// Redirector supports HTTP redirects in response encoding.
type Redirector = httpx.Redirector

// NewResponse creates a Response wrapper.
var NewResponse = httpx.NewResponse

// WrapHandler wraps http.Handler into HandlerFunc.
var WrapHandler = httpx.WrapHandler

// WrapMiddleware wraps net/http middleware into MiddlewareFunc.
var WrapMiddleware = httpx.WrapMiddleware

// Default codec helpers re-exported from httpx.
var (
	DefaultRequestVars    = httpx.DefaultRequestVars
	DefaultRequestQuery   = httpx.DefaultRequestQuery
	DefaultRequestDecoder = httpx.DefaultRequestDecoder
	DefaultResponseEncoder = httpx.DefaultResponseEncoder
	DefaultErrorEncoder   = httpx.DefaultErrorEncoder
	CodecForRequest       = httpx.CodecForRequest
)

// NewServer creates an HTTP server.
//
// Deprecated: use transport.NewHTTPServer or khttp.NewServer.
func NewServer(opts ...ServerOption) *Server {
	return &Server{Server: khttp.NewServer(opts...)}
}

// Route registers an HTTP router with next MiddlewareFunc DSL.
//
// Deprecated: use httpx.NewRouter(s.Server, prefix, m...).
func (s *Server) Route(prefix string, m ...MiddlewareFunc) Router {
	return httpx.NewRouter(s.Server, prefix, m...)
}

// Re-export kratos server options for v1 import compatibility.

// Network sets server network.
//
// Deprecated: use khttp.Network.
var Network = khttp.Network

// Address sets server listen address.
//
// Deprecated: use khttp.Address.
var Address = khttp.Address

// Endpoint sets server endpoint URL.
//
// Deprecated: use khttp.Endpoint.
var Endpoint = khttp.Endpoint

// Timeout sets server timeout.
//
// Deprecated: use khttp.Timeout.
var Timeout = khttp.Timeout

// Middleware adds service middleware.
//
// Deprecated: use khttp.Middleware.
var Middleware = khttp.Middleware

// Filter adds HTTP filters.
//
// Deprecated: use khttp.Filter.
var Filter = khttp.Filter

// RequestVarsDecoder sets vars decoder.
//
// Deprecated: use khttp.RequestVarsDecoder.
var RequestVarsDecoder = khttp.RequestVarsDecoder

// RequestQueryDecoder sets query decoder.
//
// Deprecated: use khttp.RequestQueryDecoder.
var RequestQueryDecoder = khttp.RequestQueryDecoder

// RequestDecoder sets body decoder.
//
// Deprecated: use khttp.RequestDecoder.
var RequestDecoder = khttp.RequestDecoder

// ResponseEncoder sets response encoder.
//
// Deprecated: use khttp.ResponseEncoder.
var ResponseEncoder = khttp.ResponseEncoder

// ErrorEncoder sets error encoder.
//
// Deprecated: use khttp.ErrorEncoder.
var ErrorEncoder = khttp.ErrorEncoder

// TLSConfig sets TLS config.
//
// Deprecated: use khttp.TLSConfig.
var TLSConfig = khttp.TLSConfig

// StrictSlash configures mux StrictSlash.
//
// Deprecated: use khttp.StrictSlash.
var StrictSlash = khttp.StrictSlash

// Listener sets a custom listener.
//
// Deprecated: use khttp.Listener.
var Listener = khttp.Listener

// PathPrefix sets mux path prefix.
//
// Deprecated: use khttp.PathPrefix.
var PathPrefix = khttp.PathPrefix

// NotFoundHandler sets 404 handler.
//
// Deprecated: use khttp.NotFoundHandler.
var NotFoundHandler = khttp.NotFoundHandler

// MethodNotAllowedHandler sets 405 handler.
//
// Deprecated: use khttp.MethodNotAllowedHandler.
var MethodNotAllowedHandler = khttp.MethodNotAllowedHandler

// NewHandler registers an HTTP handler on the server.
func (s *Server) NewHandler(handler Handler, middlewares ...MiddlewareFunc) {
	handler.Handle(s.Route(handler.Prefix(), middlewares...))
}

// Transporter is the HTTP transport metadata interface used by middleware/tracing.
//
// Deprecated: use khttp.Transporter.
type Transporter = khttp.Transporter

// Transport is the HTTP transport metadata carrier.
//
// Deprecated: use khttp.Transport.
type Transport = khttp.Transport

// SetOperation sets transport operation in context.
//
// Deprecated: use khttp.SetOperation.
var SetOperation = khttp.SetOperation

// RequestFromServerContext returns request from server context.
//
// Deprecated: use khttp.RequestFromServerContext.
var RequestFromServerContext = khttp.RequestFromServerContext

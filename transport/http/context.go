package http

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"github.com/nextmicro/gokit/trace"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/mux"

	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http/binding"
)

var _ Context = (*wrapper)(nil)

type CustomResponse struct {
	Code     int               `json:"code"`
	Reason   string            `json:"reason"`
	Message  string            `json:"message"`
	Data     interface{}       `json:"data"`
	TraceId  string            `json:"trace_id"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Cause    error             `json:"-"`
}

// Unwrap satisfies the Go 1.13 error wrapper interface.
func (r *CustomResponse) Unwrap() error {
	return r.Cause
}

// Context is an HTTP Context.
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
	Bind(interface{}) error
	BindVars(interface{}) error
	BindQuery(interface{}) error
	BindForm(interface{}) error
	Returns(interface{}, error) error
	Result(int, interface{}) error
	JSON(int, interface{}) error
	XML(int, interface{}) error
	String(int, string) error
	Blob(int, string, []byte) error
	Stream(int, string, io.Reader) error
	Reset(http.ResponseWriter, *http.Request)
	Context() context.Context
	Success(data ...interface{}) error
}

type responseWriter struct {
	code int
	w    http.ResponseWriter
}

func (w *responseWriter) reset(res http.ResponseWriter) {
	w.w = res
	w.code = http.StatusOK
}
func (w *responseWriter) Header() http.Header        { return w.w.Header() }
func (w *responseWriter) WriteHeader(statusCode int) { w.code = statusCode }
func (w *responseWriter) Write(data []byte) (int, error) {
	w.w.WriteHeader(w.code)
	return w.w.Write(data)
}

type wrapper struct {
	router *router
	req    *http.Request
	res    *Response
}

// NewContext returns a Context instance.
func NewContext(r *router, req *http.Request, w http.ResponseWriter) Context {
	return &wrapper{
		router: r,
		req:    req,
		res:    &Response{},
	}
}

func (c *wrapper) Header() http.Header {
	return c.req.Header
}

func (c *wrapper) Vars() url.Values {
	raws := mux.Vars(c.req)
	vars := make(url.Values, len(raws))
	for k, v := range raws {
		vars[k] = []string{v}
	}
	return vars
}

func (c *wrapper) Form() url.Values {
	if err := c.req.ParseForm(); err != nil {
		return url.Values{}
	}
	return c.req.Form
}

func (c *wrapper) Query() url.Values {
	return c.req.URL.Query()
}
func (c *wrapper) Request() *http.Request     { return c.req }
func (c *wrapper) SetRequest(r *http.Request) { c.req = r }
func (c *wrapper) Response() *Response        { return c.res }
func (c *wrapper) SetResponse(w *Response)    { c.res = w }
func (c *wrapper) Middleware(h middleware.Handler) middleware.Handler {
	if tr, ok := transport.FromServerContext(c.req.Context()); ok {
		return middleware.Chain(c.router.srv.middleware.Match(tr.Operation())...)(h)
	}
	return middleware.Chain(c.router.srv.middleware.Match(c.req.URL.Path)...)(h)
}
func (c *wrapper) Bind(v interface{}) error      { return c.router.srv.decBody(c.req, v) }
func (c *wrapper) BindVars(v interface{}) error  { return c.router.srv.decVars(c.req, v) }
func (c *wrapper) BindQuery(v interface{}) error { return c.router.srv.decQuery(c.req, v) }
func (c *wrapper) BindForm(v interface{}) error  { return binding.BindForm(c.req, v) }
func (c *wrapper) Returns(v interface{}, err error) error {
	if err != nil {
		return err
	}
	return c.router.srv.enc(c.res, c.req, v)
}

func (c *wrapper) Result(code int, v interface{}) error {
	c.res.WriteHeader(code)
	err := c.router.srv.enc(c.res, c.req, v)
	if err != nil {
		return err
	}

	return err
}

func (c *wrapper) JSON(code int, v interface{}) error {
	c.res.Header().Set("Content-Type", "application/json")
	c.res.WriteHeader(code)
	return json.NewEncoder(c.res).Encode(v)
}

func (c *wrapper) XML(code int, v interface{}) error {
	c.res.Header().Set("Content-Type", "application/xml")
	c.res.WriteHeader(code)
	return xml.NewEncoder(c.res).Encode(v)
}

func (c *wrapper) String(code int, text string) error {
	c.res.Header().Set("Content-Type", "text/plain")
	c.res.WriteHeader(code)
	_, err := c.res.Write([]byte(text))
	if err != nil {
		return err
	}
	return nil
}

func (c *wrapper) Blob(code int, contentType string, data []byte) error {
	c.res.Header().Set("Content-Type", contentType)
	c.res.WriteHeader(code)
	_, err := c.res.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func (c *wrapper) Stream(code int, contentType string, rd io.Reader) error {
	c.res.Header().Set("Content-Type", contentType)
	c.res.WriteHeader(code)
	_, err := io.Copy(c.res, rd)
	return err
}

func (c *wrapper) Reset(w http.ResponseWriter, req *http.Request) {
	c.req = req
	c.res.reset(w)
}

func (c *wrapper) Context() context.Context {
	if c.Request() == nil {
		return context.TODO()
	}
	return c.Request().Context()
}

func (c *wrapper) Deadline() (time.Time, bool) {
	if c.req == nil {
		return time.Time{}, false
	}
	return c.req.Context().Deadline()
}

func (c *wrapper) Done() <-chan struct{} {
	if c.req == nil {
		return nil
	}
	return c.req.Context().Done()
}

func (c *wrapper) Err() error {
	if c.req == nil {
		return context.Canceled
	}
	return c.req.Context().Err()
}

func (c *wrapper) Value(key interface{}) interface{} {
	if c.req == nil {
		return nil
	}
	return c.req.Context().Value(key)
}

func (c *wrapper) Success(data ...interface{}) error {
	rsp := &CustomResponse{
		Message: "success",
		TraceId: trace.ExtractTraceId(c.Context()),
	}
	if len(data) > 0 {
		rsp.Data = data[0]
	}
	return c.JSON(http.StatusOK, rsp)
}

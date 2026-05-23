package httpx

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/mux"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport/http/binding"
)

var _ Context = (*wrapper)(nil)

type wrapper struct {
	router *router
	req    *http.Request
	res    *Response
	kctx   khttp.Context
}

func newContext(r *router) Context {
	return &wrapper{router: r}
}

func (c *wrapper) Header() http.Header { return c.req.Header }

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

func (c *wrapper) Query() url.Values { return c.req.URL.Query() }

func (c *wrapper) Request() *http.Request     { return c.req }
func (c *wrapper) SetRequest(r *http.Request) { c.req = r }
func (c *wrapper) Response() *Response          { return c.res }
func (c *wrapper) SetResponse(w *Response)    { c.res = w }

func (c *wrapper) Middleware(h middleware.Handler) middleware.Handler {
	if c.kctx != nil {
		return c.kctx.Middleware(h)
	}
	return h
}

func (c *wrapper) Bind(v interface{}) error      { return c.router.opts.DecBody(c.req, v) }
func (c *wrapper) BindVars(v interface{}) error  { return c.router.opts.DecVars(c.req, v) }
func (c *wrapper) BindQuery(v interface{}) error { return c.router.opts.DecQuery(c.req, v) }
func (c *wrapper) BindForm(v interface{}) error  { return binding.BindForm(c.req, v) }

func (c *wrapper) Returns(v interface{}, err error) error {
	if err != nil {
		return err
	}
	return c.router.opts.Enc(c.res, c.req, v)
}

func (c *wrapper) Result(code int, v interface{}) error {
	c.res.WriteHeader(code)
	return c.router.opts.Enc(c.res, c.req, v)
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
	return err
}

func (c *wrapper) Blob(code int, contentType string, data []byte) error {
	c.res.Header().Set("Content-Type", contentType)
	c.res.WriteHeader(code)
	_, err := c.res.Write(data)
	return err
}

func (c *wrapper) Stream(code int, contentType string, rd io.Reader) error {
	c.res.Header().Set("Content-Type", contentType)
	c.res.WriteHeader(code)
	_, err := io.Copy(c.res, rd)
	return err
}

func (c *wrapper) Reset(w http.ResponseWriter, req *http.Request) {
	c.req = req
	if w != nil {
		if c.res == nil {
			c.res = NewResponse(w)
		} else {
			c.res.reset(w)
		}
	}
	c.kctx = nil
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

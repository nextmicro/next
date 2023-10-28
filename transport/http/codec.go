package http

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/nextmicro/logger"
	"github.com/nextmicro/next/internal/httputil"

	"github.com/gorilla/mux"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/transport/http/binding"
)

// SupportPackageIsVersion1 These constants should not be referenced from any other code.
const SupportPackageIsVersion1 = true

// Redirector replies to the request with a redirect to url
// which may be a path relative to the request path.
type Redirector interface {
	Redirect() (string, int)
}

// Request type net/http.
type Request = http.Request

// ResponseWriter type net/http.
type ResponseWriter = http.ResponseWriter

// Flusher type net/http
type Flusher = http.Flusher

// DecodeRequestFunc is decode request func.
type DecodeRequestFunc func(*http.Request, interface{}) error

// EncodeResponseFunc is encode response func.
type EncodeResponseFunc func(http.ResponseWriter, *http.Request, interface{}) error

// EncodeErrorFunc is encode error func.
type EncodeErrorFunc func(Context, error)

// DefaultRequestVars decodes the request vars to object.
func DefaultRequestVars(r *http.Request, v interface{}) error {
	raws := mux.Vars(r)
	vars := make(url.Values, len(raws))
	for k, v := range raws {
		vars[k] = []string{v}
	}
	return binding.BindQuery(vars, v)
}

// DefaultRequestQuery decodes the request vars to object.
func DefaultRequestQuery(r *http.Request, v interface{}) error {
	return binding.BindQuery(r.URL.Query(), v)
}

// DefaultRequestDecoder decodes the request body to object.
func DefaultRequestDecoder(r *http.Request, v interface{}) error {
	codec, ok := CodecForRequest(r, "Content-Type")
	if !ok {
		return errors.BadRequest("CODEC", fmt.Sprintf("unregister Content-Type: %s", r.Header.Get("Content-Type")))
	}
	data, err := io.ReadAll(r.Body)

	// reset body.
	r.Body = io.NopCloser(bytes.NewBuffer(data))

	if err != nil {
		return errors.BadRequest("CODEC", err.Error())
	}
	if len(data) == 0 {
		return nil
	}
	if err = codec.Unmarshal(data, v); err != nil {
		return errors.BadRequest("CODEC", fmt.Sprintf("body unmarshal %s", err.Error()))
	}
	return nil
}

// DefaultResponseEncoder encodes the object to the HTTP response.
func DefaultResponseEncoder(w http.ResponseWriter, r *http.Request, v interface{}) error {
	if v == nil {
		return nil
	}
	if rd, ok := v.(Redirector); ok {
		uri, code := rd.Redirect()
		http.Redirect(w, r, uri, code)
		return nil
	}

	rsp := &CustomResponse{
		Code:    0,
		Reason:  "OK",
		Message: "success",
		Data:    v,
		TraceId: w.Header().Get("x-trace-id"),
	}
	codec, _ := CodecForRequest(r, "Accept")
	data, err := codec.Marshal(rsp)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", httputil.ContentType(codec.Name()))
	_, err = w.Write(data)
	return err
}

// DefaultErrorEncoder encodes the error to the HTTP response.
func DefaultErrorEncoder(c Context, err error) {
	response := &CustomResponse{
		Code:     http.StatusInternalServerError,
		Reason:   "UNKNOWN_REASON",
		Message:  "服务内部错误",
		Metadata: make(map[string]string),
		TraceId:  c.Response().Header().Get("x-trace-id"),
	}

	switch errType := err.(type) {
	case *errors.Error:
		response.Cause = errType.Unwrap()
		response.Code = int(errType.Code)
		response.Reason = errType.Reason
		response.Message = errType.Message
		response.Metadata = errType.Metadata
	default:
		se := errors.FromError(err)
		response.Cause = se.Unwrap()
		response.Code = int(se.GetCode())
		response.Message = se.GetMessage()
		response.Metadata = se.GetMetadata()
	}

	// Send response
	if c.Request().Method == http.MethodHead { // Issue #608
		c.Response().WriteHeader(http.StatusOK)
	} else {
		codec, _ := CodecForRequest(c.Request(), "Accept")
		body, err := codec.Marshal(response)
		if err != nil {
			c.Response().WriteHeader(http.StatusInternalServerError)
			return
		}

		c.Response().Header().Set("Content-Type", httputil.ContentType(codec.Name()))
		c.Response().WriteHeader(response.Code)
		_, _ = c.Response().Write(body)
	}

	if response.Unwrap() != nil {
		logger.WithContext(c.Request().Context()).Error(response.Unwrap())
	}
}

// CodecForRequest get encoding.Codec via http.Request
func CodecForRequest(r *http.Request, name string) (encoding.Codec, bool) {
	for _, accept := range r.Header[name] {
		codec := encoding.GetCodec(httputil.ContentSubtype(accept))
		if codec != nil {
			return codec, true
		}
	}
	return encoding.GetCodec("json"), false
}

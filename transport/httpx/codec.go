package httpx

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"
	"github.com/nextmicro/next/internal/httputil"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/transport/http/binding"
)

// Redirector replies to the request with a redirect to url.
type Redirector interface {
	Redirect() (string, int)
}

// DecodeRequestFunc decodes the request into v.
type DecodeRequestFunc = khttp.DecodeRequestFunc

// EncodeResponseFunc encodes v into the HTTP response.
type EncodeResponseFunc = khttp.EncodeResponseFunc

// EncodeErrorFunc encodes an error into the HTTP response.
type EncodeErrorFunc func(Context, error)

// DefaultRequestVars decodes the request vars to object.
func DefaultRequestVars(r *http.Request, v interface{}) error {
	raws := mux.Vars(r)
	vars := make(url.Values, len(raws))
	for k, val := range raws {
		vars[k] = []string{val}
	}
	return binding.BindQuery(vars, v)
}

// DefaultRequestQuery decodes the request query to object.
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

// DefaultResponseEncoder encodes the object to the HTTP response using next envelope format.
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

// CodecForRequest gets encoding.Codec via http.Request Accept/Content-Type headers.
func CodecForRequest(r *http.Request, name string) (encoding.Codec, bool) {
	for _, accept := range r.Header[name] {
		codec := encoding.GetCodec(httputil.ContentSubtype(accept))
		if codec != nil {
			return codec, true
		}
	}
	return encoding.GetCodec("json"), false
}

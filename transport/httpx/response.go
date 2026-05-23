package httpx

import (
	"bufio"
	"net"
	"net/http"
)

// Response wraps an http.ResponseWriter and implements its interface to be used
// by an HTTP handler to construct an HTTP response.
type Response struct {
	Writer http.ResponseWriter
	Status int
	Size   int64
}

// NewResponse creates a new instance of Response.
func NewResponse(w http.ResponseWriter) *Response {
	return &Response{
		Writer: w,
		Status: http.StatusOK,
	}
}

// Header returns the header map for the writer that will be sent by WriteHeader.
func (r *Response) Header() http.Header {
	return r.Writer.Header()
}

// WriteHeader sends an HTTP response header with status code.
func (r *Response) WriteHeader(code int) {
	r.Status = code
}

// StatusCode returns the status code, defaulting to 200.
func (r *Response) StatusCode() int {
	if r.Status == 0 {
		r.Status = http.StatusOK
	}
	return r.Status
}

// Unwrap returns the underlying http.ResponseWriter.
func (r *Response) Unwrap() http.ResponseWriter { return r.Writer }

// Write writes the data to the connection as part of an HTTP reply.
func (r *Response) Write(b []byte) (n int, err error) {
	r.Writer.WriteHeader(r.StatusCode())
	n, err = r.Writer.Write(b)
	r.Size += int64(n)
	return
}

// Flush implements the http.Flusher interface.
func (r *Response) Flush() {
	r.Writer.(http.Flusher).Flush()
}

// Hijack implements the http.Hijacker interface.
func (r *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return r.Writer.(http.Hijacker).Hijack()
}

func (r *Response) reset(w http.ResponseWriter) {
	r.Writer = w
	r.Size = 0
	r.Status = http.StatusOK
}

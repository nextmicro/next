package httpx

import (
	"net/http"

	"github.com/nextmicro/logger"
	"github.com/nextmicro/next/internal/httputil"

	"github.com/go-kratos/kratos/v2/errors"
)

// CustomResponse is the next standard HTTP response envelope.
type CustomResponse struct {
	Code     int               `json:"code"`
	Reason   string            `json:"reason,omitempty"`
	Message  string            `json:"message"`
	Data     interface{}       `json:"data,omitempty"`
	TraceId  string            `json:"trace_id,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Cause    error             `json:"-"`
}

// Unwrap satisfies the Go 1.13 error wrapper interface.
func (r *CustomResponse) Unwrap() error { return r.Cause }

// DefaultErrorEncoder encodes the error to the HTTP response using next envelope format.
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

	if c.Request().Method == http.MethodHead {
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

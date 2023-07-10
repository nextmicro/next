package metrics

import (
	"github.com/go-kratos/kratos/v2/errors"
	"go.opentelemetry.io/otel/codes"
	"net/http"
)

type Converter interface {
	ToCode(code int) codes.Code
	FromErrorCode(err error) codes.Code
}

type statusConverter struct{}

// DefaultConverter default converter.
var DefaultConverter Converter = statusConverter{}

// ToCode converts a HTTP error code into the corresponding metrics response code.
func (c statusConverter) ToCode(code int) codes.Code {
	if code >= http.StatusContinue && code < http.StatusInternalServerError {
		return codes.Ok
	} else if code >= http.StatusInternalServerError && code < http.StatusNetworkAuthenticationRequired {
		return codes.Error
	}
	return codes.Unset
}

// FromErrorCode converts an error code into the corresponding metrics response code.
func (c statusConverter) FromErrorCode(err error) codes.Code {
	if err == nil {
		return codes.Ok
	}

	se := errors.FromError(err)
	return c.ToCode(int(se.Code))
}

// ToCode converts a HTTP error code into the corresponding metrics response code.
func ToCode(code int) codes.Code {
	return DefaultConverter.ToCode(code)
}

// FromErrorCode converts an error code into the corresponding metrics response code.
func FromErrorCode(err error) codes.Code {
	return DefaultConverter.FromErrorCode(err)
}

package metrics

import (
	"github.com/go-kratos/kratos/v2/errors"
	"go.opentelemetry.io/otel/codes"
	"net/http"
	"testing"
)

func TestToCode(t *testing.T) {
	tests := []struct {
		name string
		code int
		want codes.Code
	}{
		{"http.StatusOK", http.StatusOK, codes.Ok},
		{"http.StatusBadRequest", http.StatusBadRequest, codes.Ok},
		{"http.StatusUnauthorized", http.StatusUnauthorized, codes.Ok},
		{"http.StatusForbidden", http.StatusForbidden, codes.Ok},
		{"http.StatusNotFound", http.StatusNotFound, codes.Ok},
		{"http.StatusConflict", http.StatusConflict, codes.Ok},
		{"http.StatusTooManyRequests", http.StatusTooManyRequests, codes.Ok},
		{"http.StatusInternalServerError", http.StatusInternalServerError, codes.Error},
		{"http.StatusNotImplemented", http.StatusNotImplemented, codes.Error},
		{"http.StatusServiceUnavailable", http.StatusServiceUnavailable, codes.Error},
		{"http.StatusGatewayTimeout", http.StatusGatewayTimeout, codes.Error},
		{"else", 100000, codes.Unset},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToCode(tt.code); got != tt.want {
				t.Errorf("GRPCCodeFromStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromErrorCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{
			name: "BadRequest",
			err:  errors.BadRequest("reason_400", "message_400"),
			want: codes.Ok,
		},
		{
			name: "Unauthorized",
			err:  errors.Unauthorized("reason_401", "message_401"),
			want: codes.Ok,
		},
		{
			name: "Forbidden",
			err:  errors.Forbidden("reason_403", "message_403"),
			want: codes.Ok,
		},
		{
			name: "NotFound",
			err:  errors.NotFound("reason_404", "message_404"),
			want: codes.Ok,
		},
		{
			name: "Conflict",
			err:  errors.Conflict("reason_409", "message_409"),
			want: codes.Ok,
		},
		{
			name: "InternalServer",
			err:  errors.InternalServer("reason_500", "message_500"),
			want: codes.Error,
		},
		{
			name: "ServiceUnavailable",
			err:  errors.ServiceUnavailable("reason_503", "message_503"),
			want: codes.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FromErrorCode(tt.err); got != tt.want {
				t.Errorf("StatusFromGRPCCode() = %v, want %v", got, tt.want)
			}
		})
	}
}

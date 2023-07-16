package http

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	HeaderServer        = "Server"
	HeaderXFrameOptions = "X-Frame-Options"
)

func TestResponse(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := NewContext(testRouter, req, rec)
	res := &Response{Writer: rec}

	// Before
	res.Before(func() {
		c.Response().Header().Set(HeaderServer, "echo")
	})
	// After
	res.After(func() {
		c.Response().Header().Set(HeaderXFrameOptions, "DENY")
	})
	res.Write([]byte("test"))
	assert.Equal(t, "echo", rec.Header().Get(HeaderServer))
	assert.Equal(t, "DENY", rec.Header().Get(HeaderXFrameOptions))
}

func TestResponse_Write_FallsBackToDefaultStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	res := &Response{Writer: rec}

	res.Write([]byte("test"))
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestResponse_Write_UsesSetResponseCode(t *testing.T) {
	rec := httptest.NewRecorder()
	res := &Response{Writer: rec}

	res.Status = http.StatusBadRequest
	res.Write([]byte("test"))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestResponse_Flush(t *testing.T) {
	rec := httptest.NewRecorder()
	res := &Response{Writer: rec}

	res.Write([]byte("test"))
	res.Flush()
	assert.True(t, rec.Flushed)
}

func TestResponse_ChangeStatusCodeBeforeWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	res := &Response{Writer: rec}

	res.Before(func() {
		if 200 < res.Status && res.Status < 300 {
			res.Status = 200
		}
	})

	res.WriteHeader(209)
	res.Write([]byte("test"))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "test", rec.Body.String())
}

func TestResponse_Header(t *testing.T) {
	rec := httptest.NewRecorder()
	res := &Response{Writer: rec}
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)

	err := DefaultResponseEncoder(res, req, map[string]interface{}{
		"foo": "bar",
	})
	assert.NoError(t, err)

	if res.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected %v, got %v", "application/json", res.Header().Get("Content-Type"))
	}
}

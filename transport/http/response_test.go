package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
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

	c.Response().Header().Set(HeaderServer, "echo")
	c.Response().Header().Set(HeaderXFrameOptions, "DENY")
	n, err := res.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, "echo", rec.Header().Get(HeaderServer))
	assert.Equal(t, "DENY", rec.Header().Get(HeaderXFrameOptions))
}

func TestResponse_Write_FallsBackToDefaultStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	res := &Response{Writer: rec}

	n, err := res.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestResponse_Write_UsesSetResponseCode(t *testing.T) {
	rec := httptest.NewRecorder()
	res := &Response{Writer: rec}

	res.Status = http.StatusBadRequest
	n, err := res.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestResponse_Flush(t *testing.T) {
	rec := httptest.NewRecorder()
	res := &Response{Writer: rec}

	n, err := res.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	res.Flush()
	assert.True(t, rec.Flushed)
}

func TestResponse_ChangeStatusCodeBeforeWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	res := &Response{Writer: rec}

	res.WriteHeader(209)
	n, err := res.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)

	assert.Equal(t, 209, rec.Code)
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

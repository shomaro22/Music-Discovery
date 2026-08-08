package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"music-discovery/backend/entity"

	"github.com/labstack/echo/v4"
)

func TestHTTPErrorHandlerCommonFormat(t *testing.T) {
	tests := []struct {
		status int
		code   string
	}{
		{400, "BAD_REQUEST"}, {404, "NOT_FOUND"}, {405, "METHOD_NOT_ALLOWED"},
		{413, "PAYLOAD_TOO_LARGE"}, {415, "UNSUPPORTED_MEDIA_TYPE"}, {429, "RATE_LIMIT_EXCEEDED"},
	}
	for _, tt := range tests {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Response().Header().Set(echo.HeaderXRequestID, "req-1")
		NewHTTPErrorHandler(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))(echo.NewHTTPError(tt.status), c)
		var got apiErrorResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if rec.Code != tt.status || got.Code != tt.code || got.RequestID != "req-1" {
			t.Fatalf("status=%d got=%#v", rec.Code, got)
		}
		decoder := json.NewDecoder(bytes.NewReader(rec.Body.Bytes()))
		decoder.DisallowUnknownFields()
		var exactResponse apiErrorResponse
		if err := decoder.Decode(&exactResponse); err != nil {
			t.Fatalf("unexpected response field: %v body=%s", err, rec.Body.String())
		}
	}
}

func TestHTTPErrorHandlerDoesNotExposeInternalError(t *testing.T) {
	var logBuffer bytes.Buffer
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	NewHTTPErrorHandler(slog.New(slog.NewTextHandler(&logBuffer, nil)))(errors.New("database password=secret"), c)
	if bytes.Contains(rec.Body.Bytes(), []byte("secret")) {
		t.Fatalf("secret leaked: %s", rec.Body.String())
	}
	var got apiErrorResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Code != "INTERNAL_ERROR" || rec.Code != 500 {
		t.Fatalf("got=%#v status=%d", got, rec.Code)
	}
}

func TestHTTPErrorHandlerReturnsUserSuspendedCode(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/recommendation-history", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	NewHTTPErrorHandler(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))(
		echo.NewHTTPError(http.StatusForbidden, entity.ErrUserSuspended),
		c,
	)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status=%d", rec.Code)
	}
	var body apiErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != "USER_SUSPENDED" {
		t.Fatalf("code=%s body=%s", body.Code, rec.Body.String())
	}
}

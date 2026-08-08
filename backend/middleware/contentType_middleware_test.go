package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRequireJSON(t *testing.T) {
	tests := []struct {
		contentType string
		allowed     bool
	}{
		{echo.MIMEApplicationJSON, true},
		{"application/json; charset=utf-8", true},
		{"application/json-invalid", false},
		{"text/plain", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.contentType != "" {
				req.Header.Set(echo.HeaderContentType, tt.contentType)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			called := false
			err := RequireJSON(func(echo.Context) error { called = true; return nil })(c)
			if tt.allowed && (err != nil || !called) {
				t.Fatalf("expected allowed, err=%v", err)
			}
			if !tt.allowed && err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

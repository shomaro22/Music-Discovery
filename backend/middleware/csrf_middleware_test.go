package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"music-discovery/backend/entity"

	"github.com/labstack/echo/v4"
)

func runCSRF(t *testing.T, cookieValue, headerValue string) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: "csrf_token", Value: cookieValue})
	}
	if headerValue != "" {
		req.Header.Set("X-CSRF-Token", headerValue)
	}
	rec := httptest.NewRecorder()
	called := false
	handler := CSRF(func(c echo.Context) error { called = true; return c.NoContent(http.StatusNoContent) })
	if err := handler(e.NewContext(req, rec)); err != nil {
		t.Fatal(err)
	}
	return rec, called
}

func TestCSRFMiddlewareMatch(t *testing.T) {
	rec, called := runCSRF(t, "same-token", "same-token")
	if rec.Code != http.StatusNoContent || !called {
		t.Fatalf("status=%d called=%v", rec.Code, called)
	}
}

func TestCSRFMiddlewareRejectsMissingOrMismatch(t *testing.T) {
	tests := []struct{ name, cookie, header string }{
		{name: "cookie missing", header: "token"},
		{name: "header missing", cookie: "token"},
		{name: "mismatch", cookie: "cookie", header: "header"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, called := runCSRF(t, tt.cookie, tt.header)
			if rec.Code != http.StatusForbidden || called {
				t.Fatalf("status=%d called=%v", rec.Code, called)
			}
			var apiErr entity.APIErrorResponse
			if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
				t.Fatal(err)
			}
			if apiErr.Code != "CSRF_TOKEN_INVALID" {
				t.Fatalf("code=%q", apiErr.Code)
			}
		})
	}
}

package middleware

import (
	"github.com/labstack/echo/v4"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCSRFMiddleware(t *testing.T) {
	e := echo.New()
	next := func(c echo.Context) error { return c.NoContent(204) }
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: "same"})
	req.Header.Set("X-CSRF-Token", "same")
	rec := httptest.NewRecorder()
	if err := NewCSRFMiddleware("csrf_token").Validate(next)(e.NewContext(req, rec)); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 204 {
		t.Fatalf("status=%d", rec.Code)
	}
}

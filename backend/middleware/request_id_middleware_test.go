package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestRequestIDMiddlewareGeneratesAndReturnsID(t *testing.T) {
	e := echo.New()
	handler := NewRequestIDMiddleware()(func(ctx echo.Context) error { return ctx.NoContent(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	if err := handler(ctx); err != nil {
		t.Fatal(err)
	}
	if got := rec.Header().Get(echo.HeaderXRequestID); !requestIDPattern.MatchString(got) {
		t.Fatalf("invalid generated request id: %q", got)
	}
}

func TestRequestIDMiddlewareRejectsInvalidIncomingID(t *testing.T) {
	e := echo.New()
	handler := NewRequestIDMiddleware()(func(ctx echo.Context) error { return ctx.NoContent(http.StatusNoContent) })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(echo.HeaderXRequestID, "bad\nvalue")
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	if err := handler(ctx); err != nil {
		t.Fatal(err)
	}
	if got := rec.Header().Get(echo.HeaderXRequestID); got == "bad\nvalue" || !requestIDPattern.MatchString(got) {
		t.Fatalf("unexpected request id: %q", got)
	}
}

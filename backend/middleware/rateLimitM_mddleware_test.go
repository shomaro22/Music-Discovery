package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

func TestSignupRateLimitAndRetryAfter(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	e := echo.New()
	handler := NewRateLimitMiddleware(client).SignUp(func(c echo.Context) error { return c.NoContent(http.StatusNoContent) })

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", nil)
		req.RemoteAddr = "127.0.0.1:1234"
		rec := httptest.NewRecorder()
		if err := handler(e.NewContext(req, rec)); err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
		if rec.Code != http.StatusNoContent {
			t.Fatalf("request %d status=%d", i+1, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	err := handler(e.NewContext(req, rec))
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusTooManyRequests {
		t.Fatalf("err=%v", err)
	}
	retryAfter, parseErr := strconv.Atoi(rec.Header().Get("Retry-After"))
	if parseErr != nil || retryAfter < 1 {
		t.Fatalf("Retry-After=%q", rec.Header().Get("Retry-After"))
	}
}

func TestSignupRateLimitUsesRequestContext(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	e := echo.New()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", nil).WithContext(ctx)
	err := NewRateLimitMiddleware(client).SignUp(func(c echo.Context) error { return nil })(e.NewContext(req, httptest.NewRecorder()))
	if err == nil {
		t.Fatal("expected canceled context error")
	}
}

package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"music-discovery/backend/repository"
	"music-discovery/backend/usecase"

	"github.com/alicebob/miniredis/v2"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

func TestSignupRateLimitAndRetryAfter(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	e := echo.New()
	handler := NewRateLimitMiddleware(usecase.NewRateLimitUsecase(repository.NewRateLimitRepository(client), middlewareTestClock{})).SignUp(func(c echo.Context) error { return c.NoContent(http.StatusNoContent) })

	for i := 0; i < 10; i++ {
		body := bytes.NewBufferString(`{"email":"user` + strconv.Itoa(i) + `@example.com"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", body)
		req.RemoteAddr = "127.0.0.1:1234"
		rec := httptest.NewRecorder()
		if err := handler(e.NewContext(req, rec)); err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
		if rec.Code != http.StatusNoContent {
			t.Fatalf("request %d status=%d", i+1, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(`{"email":"overflow@example.com"}`))
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
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewBufferString(`{"email":"user@example.com"}`)).WithContext(ctx)
	err := NewRateLimitMiddleware(usecase.NewRateLimitUsecase(repository.NewRateLimitRepository(client), middlewareTestClock{})).SignUp(func(c echo.Context) error { return nil })(e.NewContext(req, httptest.NewRecorder()))
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %v", err)
	}
}

type middlewareTestClock struct{}

func (middlewareTestClock) Now() time.Time { return time.Now().UTC() }

func TestRefreshRateLimitAndRetryAfter(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	e := echo.New()
	called := 0
	handler := NewRateLimitMiddleware(usecase.NewRateLimitUsecase(repository.NewRateLimitRepository(client), middlewareTestClock{})).Refresh(func(c echo.Context) error { called++; return c.NoContent(http.StatusNoContent) })
	for i := 0; i < 30; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
		req.RemoteAddr = "198.51.100.10:1234"
		rec := httptest.NewRecorder()
		if err := handler(e.NewContext(req, rec)); err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
		if rec.Code != http.StatusNoContent {
			t.Fatalf("request %d status=%d", i+1, rec.Code)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req.RemoteAddr = "198.51.100.10:1234"
	rec := httptest.NewRecorder()
	err := handler(e.NewContext(req, rec))
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusTooManyRequests {
		t.Fatalf("err=%v", err)
	}
	if rec.Header().Get("X-RateLimit-Name") != "refresh_ip" {
		t.Fatalf("limit name=%q", rec.Header().Get("X-RateLimit-Name"))
	}
	if retryAfter, parseErr := strconv.Atoi(rec.Header().Get("Retry-After")); parseErr != nil || retryAfter < 1 {
		t.Fatalf("Retry-After=%q", rec.Header().Get("Retry-After"))
	}
	if called != 30 {
		t.Fatalf("called=%d", called)
	}
}

func TestLoginRateLimitUsesEmailAcrossDifferentIPs(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	e := echo.New()
	handler := NewRateLimitMiddleware(usecase.NewRateLimitUsecase(repository.NewRateLimitRepository(client), middlewareTestClock{})).Login(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"email":" User@Example.COM "}`))
		req.RemoteAddr = "192.0.2." + strconv.Itoa(i+1) + ":1234"
		rec := httptest.NewRecorder()
		if err := handler(e.NewContext(req, rec)); err != nil {
			t.Fatalf("request %d: %v", i+1, err)
		}
	}

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(`{"email":"user@example.com"}`))
	req.RemoteAddr = "198.51.100.20:1234"
	rec := httptest.NewRecorder()
	err := handler(e.NewContext(req, rec))
	httpErr, ok := err.(*echo.HTTPError)
	if !ok || httpErr.Code != http.StatusTooManyRequests {
		t.Fatalf("err=%v", err)
	}
	if got := rec.Header().Get("X-RateLimit-Name"); got != "login_email" {
		t.Fatalf("X-RateLimit-Name=%q", got)
	}
}

func TestRateLimitMiddlewareRestoresAuthRequestBody(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	e := echo.New()
	const body = `{"email":"user@example.com","password":"secret"}`
	handler := NewRateLimitMiddleware(usecase.NewRateLimitUsecase(repository.NewRateLimitRepository(client), middlewareTestClock{})).Login(func(c echo.Context) error {
		got, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		if string(got) != body {
			t.Fatalf("body=%q", string(got))
		}
		return c.NoContent(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	req.RemoteAddr = "203.0.113.20:1234"
	if err := handler(e.NewContext(req, httptest.NewRecorder())); err != nil {
		t.Fatal(err)
	}
}

package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

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

func TestLoginRateLimitAndIPIsolation(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cfg := RateLimitConfig{Login: RateLimitRule{Capacity: 20, RefillPerSecond: 20.0 / 60.0, TTL: 2 * time.Minute}, SignUp: RateLimitRule{Capacity: 10, RefillPerSecond: 10.0 / 60.0, TTL: 2 * time.Minute}}
	calls := 0
	handler := NewRateLimitMiddleware(client, cfg).Login(func(c echo.Context) error { calls++; return c.NoContent(http.StatusNoContent) })
	e := echo.New()
	request := func(ip string) (*httptest.ResponseRecorder, error) {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
		req.RemoteAddr = ip + ":1234"
		rec := httptest.NewRecorder()
		return rec, handler(e.NewContext(req, rec))
	}
	for i := 0; i < 20; i++ {
		rec, err := request("127.0.0.1")
		if err != nil || rec.Code != 204 {
			t.Fatalf("request=%d status=%d err=%v", i+1, rec.Code, err)
		}
	}
	rec, err := request("127.0.0.1")
	he, ok := err.(*echo.HTTPError)
	if !ok || he.Code != 429 || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("err=%v retry=%q", err, rec.Header().Get("Retry-After"))
	}
	rec, err = request("127.0.0.2")
	if err != nil || rec.Code != 204 {
		t.Fatalf("isolated status=%d err=%v", rec.Code, err)
	}
	if calls != 21 {
		t.Fatalf("next calls=%d", calls)
	}
}

func TestLoginRedisFailureContinuesAndLogs(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buffer, nil))
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 10 * time.Millisecond, ReadTimeout: 10 * time.Millisecond, WriteTimeout: 10 * time.Millisecond})
	defer client.Close()
	cfg := RateLimitConfig{Login: RateLimitRule{Capacity: 20, RefillPerSecond: 1, TTL: time.Minute}, SignUp: RateLimitRule{Capacity: 10, RefillPerSecond: 1, TTL: time.Minute}}
	called := false
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	rec := httptest.NewRecorder()
	rec.Header().Set(echo.HeaderXRequestID, "req-rate")
	err := NewRateLimitMiddleware(client, logger, cfg).Login(func(c echo.Context) error { called = true; return c.NoContent(204) })(e.NewContext(req, rec))
	if err != nil || !called {
		t.Fatalf("err=%v called=%v", err, called)
	}
	logText := buffer.String()
	for _, value := range []string{"rate_limit_redis_error", "login", "req-rate"} {
		if !strings.Contains(logText, value) {
			t.Fatalf("log=%s", logText)
		}
	}
}

func TestLoginCanceledContextDoesNotReachNext(t *testing.T) {
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil).WithContext(ctx)
	called := false
	err := NewRateLimitMiddleware(client).Login(func(c echo.Context) error { called = true; return nil })(echo.New().NewContext(req, httptest.NewRecorder()))
	if err == nil || called {
		t.Fatalf("err=%v called=%v", err, called)
	}
}

func TestRateLimitConfigValidation(t *testing.T) {
	t.Setenv("RATE_LIMIT_LOGIN_CAPACITY", "0")
	if _, err := LoadRateLimitConfigFromEnv(); err == nil {
		t.Fatal("expected invalid config")
	}
}

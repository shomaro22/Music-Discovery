package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestRoleMiddleware(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	middleware, err := NewRoleMiddleware(logger, entity.RoleContentAdmin, entity.RoleSystemAdmin)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("allows one of multiple roles", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(AuthUserIDKey, uuid.New())
		c.Set(AuthUserRoleKey, entity.RoleSystemAdmin)
		called := false
		err := middleware.Authorize(func(echo.Context) error { called = true; return nil })(c)
		if err != nil || !called {
			t.Fatalf("err=%v called=%v", err, called)
		}
	})

	t.Run("forbids unmatched role without calling handler", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		c.Set(AuthUserIDKey, uuid.New())
		c.Set(AuthUserRoleKey, entity.RoleUser)
		called := false
		err := middleware.Authorize(func(echo.Context) error { called = true; return nil })(c)
		httpErr, ok := err.(*echo.HTTPError)
		if !ok || httpErr.Code != http.StatusForbidden || called {
			t.Fatalf("err=%v called=%v", err, called)
		}
	})
}

func TestNewRoleMiddlewareRejectsInvalidConfiguration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if _, err := NewRoleMiddleware(logger); err == nil {
		t.Fatal("empty roles must fail")
	}
	if _, err := NewRoleMiddleware(logger, entity.RoleGuest); err == nil {
		t.Fatal("guest must fail")
	}
}

func TestRoleMiddlewareRejectsMissingOrInvalidContext(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	roleMiddleware, err := NewRoleMiddleware(logger, entity.RoleSystemAdmin)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		setContext func(echo.Context)
	}{
		{name: "missing user id", setContext: func(c echo.Context) { c.Set(AuthUserRoleKey, entity.RoleSystemAdmin) }},
		{name: "missing role", setContext: func(c echo.Context) { c.Set(AuthUserIDKey, uuid.New()) }},
		{name: "wrong role type", setContext: func(c echo.Context) { c.Set(AuthUserIDKey, uuid.New()); c.Set(AuthUserRoleKey, "system_admin") }},
		{name: "undefined role", setContext: func(c echo.Context) { c.Set(AuthUserIDKey, uuid.New()); c.Set(AuthUserRoleKey, entity.Role("unknown")) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
			tt.setContext(c)
			called := false
			err := roleMiddleware.Authorize(func(echo.Context) error { called = true; return nil })(c)
			httpErr, ok := err.(*echo.HTTPError)
			if !ok || httpErr.Code != http.StatusUnauthorized || called {
				t.Fatalf("err=%v called=%v", err, called)
			}
		})
	}
}

package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

func TestAccessLoggerDoesNotLogRequestBodyOrSensitiveHeaders(t *testing.T) {
	var output bytes.Buffer
	e := echo.New()
	e.Use(echoMiddleware.LoggerWithConfig(echoMiddleware.LoggerConfig{
		Format: "${method} ${uri} ${status}\n",
		Output: &output,
	}))
	e.POST("/api/auth/signup", func(c echo.Context) error { return c.NoContent(http.StatusCreated) })

	secret := `{"email":"user@example.com","password":"super-secret"}`
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", strings.NewReader(secret))
	req.Header.Set(echo.HeaderAuthorization, "Bearer access-secret")
	req.Header.Set(echo.HeaderCookie, "refresh_token=refresh-secret")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	logged := output.String()
	for _, forbidden := range []string{"super-secret", "access-secret", "refresh-secret", "password"} {
		if strings.Contains(logged, forbidden) {
			t.Fatalf("sensitive value logged: %q in %q", forbidden, logged)
		}
	}
}

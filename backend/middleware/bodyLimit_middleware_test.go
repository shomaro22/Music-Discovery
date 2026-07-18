package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
)

func TestSignupBodyLimitAndCORS(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = NewHTTPErrorHandler(slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	e.Use(echoMiddleware.RequestID())
	e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{AllowOrigins: []string{"http://localhost:5173"}}))
	e.POST("/api/auth/signup", func(c echo.Context) error { return c.NoContent(http.StatusCreated) }, echoMiddleware.BodyLimit("16K"))
	e.POST("/api/upload", func(c echo.Context) error { return c.NoContent(http.StatusCreated) })

	large := bytes.Repeat([]byte("a"), 17*1024)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", bytes.NewReader(large))
	req.Header.Set(echo.HeaderOrigin, "http://localhost:5173")
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get(echo.HeaderAccessControlAllowOrigin) != "http://localhost:5173" {
		t.Fatalf("missing CORS header: %#v", rec.Header())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("PAYLOAD_TOO_LARGE")) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/upload", bytes.NewReader(large))
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("non-signup route was limited: %d", rec.Code)
	}
}

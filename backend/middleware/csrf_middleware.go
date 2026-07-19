package middleware

import (
	"crypto/subtle"
	"net/http"
	"os"

	"music-discovery/backend/entity"

	"github.com/labstack/echo/v4"
)

func CSRF(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookieName := os.Getenv("CSRF_COOKIE_NAME")
		if cookieName == "" {
			cookieName = "csrf_token"
		}
		cookie, err := c.Cookie(cookieName)
		header := c.Request().Header.Get("X-CSRF-Token")
		if err != nil || cookie.Value == "" || header == "" || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) != 1 {
			return c.JSON(http.StatusForbidden, entity.APIErrorResponse{
				Status: http.StatusForbidden, Code: "CSRF_TOKEN_MISMATCH",
				Message: "CSRF Tokenが無効です", RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
			})
		}
		return next(c)
	}
}

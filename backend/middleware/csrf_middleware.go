package middleware

import (
	"crypto/subtle"
	"net/http"

	"music-discovery/backend/entity"

	"github.com/labstack/echo/v4"
)

func NewCSRFMiddleware(cookieName string) echo.MiddlewareFunc {
	if cookieName == "" {
		cookieName = "csrf_token"
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cookie, err := c.Cookie(cookieName)
			header := c.Request().Header.Get("X-CSRF-Token")
			if err != nil || cookie.Value == "" || header == "" || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) != 1 {
				return c.JSON(http.StatusForbidden, entity.APIErrorResponse{
					Status: http.StatusForbidden, Code: "CSRF_TOKEN_INVALID",
					Message: "CSRF Tokenが無効です", RequestID: c.Response().Header().Get(echo.HeaderXRequestID),
				})
			}
			return next(c)
		}
	}
}

// CSRF remains for compatibility; production wiring should use NewCSRFMiddleware.
func CSRF(next echo.HandlerFunc) echo.HandlerFunc {
	return NewCSRFMiddleware("csrf_token")(next)
}

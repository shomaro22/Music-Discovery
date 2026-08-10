package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func NewCSRFMiddleware(cookieName string, allowedOrigins ...string) echo.MiddlewareFunc {
	if cookieName == "" {
		cookieName = "csrf_token"
	}
	trusted := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			trusted[origin] = struct{}{}
		}
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if origin := c.Request().Header.Get(echo.HeaderOrigin); origin != "" && len(trusted) > 0 {
				if _, ok := trusted[origin]; !ok {
					return c.JSON(http.StatusForbidden, apiErrorResponse{Status: http.StatusForbidden, Code: "CSRF_ORIGIN_INVALID", Message: "Originが許可されていません", RequestID: c.Response().Header().Get(echo.HeaderXRequestID)})
				}
			}
			cookie, err := c.Cookie(cookieName)
			header := c.Request().Header.Get("X-CSRF-Token")
			if err != nil || cookie.Value == "" || header == "" || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) != 1 {
				return c.JSON(http.StatusForbidden, apiErrorResponse{
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

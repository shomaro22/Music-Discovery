package middleware

import (
	"crypto/subtle"
	"github.com/labstack/echo/v4"
	"music-discovery/backend/entity"
	"net/http"
)

type ICSRFMiddleware interface {
	Validate(next echo.HandlerFunc) echo.HandlerFunc
}
type csrfMiddleware struct{ cookieName string }

func NewCSRFMiddleware(name string) ICSRFMiddleware { return &csrfMiddleware{cookieName: name} }
func (m *csrfMiddleware) Validate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cookie, err := c.Cookie(m.cookieName)
		header := c.Request().Header.Get("X-CSRF-Token")
		if err != nil || cookie.Value == "" || header == "" || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) != 1 {
			return c.JSON(http.StatusForbidden, entity.APIErrorResponse{Status: 403, Code: "CSRF_TOKEN_INVALID", Message: "CSRF Tokenが正しくありません", RequestID: c.Response().Header().Get(echo.HeaderXRequestID)})
		}
		return next(c)
	}
}

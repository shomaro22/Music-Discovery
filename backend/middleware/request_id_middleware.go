package middleware

import (
	"regexp"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{15,99}$`)

func NewRequestIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			requestID := ctx.Request().Header.Get(echo.HeaderXRequestID)
			if !requestIDPattern.MatchString(requestID) {
				requestID = uuid.NewString()
			}
			ctx.Set(echo.HeaderXRequestID, requestID)
			ctx.SetRequest(ctx.Request().WithContext(withRequestID(ctx.Request().Context(), requestID)))
			ctx.Request().Header.Set(echo.HeaderXRequestID, requestID)
			ctx.Response().Header().Set(echo.HeaderXRequestID, requestID)
			return next(ctx)
		}
	}
}

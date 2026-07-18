package middleware

import (
	"mime"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

func RequireJSON(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		contentType := strings.TrimSpace(
			c.Request().Header.Get(echo.HeaderContentType),
		)

		if contentType == "" {
			return echo.NewHTTPError(
				http.StatusUnsupportedMediaType,
			)
		}

		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil || mediaType != echo.MIMEApplicationJSON {
			return echo.NewHTTPError(
				http.StatusUnsupportedMediaType,
			)
		}

		return next(c)
	}
}

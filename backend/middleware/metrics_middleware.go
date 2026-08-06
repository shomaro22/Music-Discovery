package middleware

import (
	"errors"
	"net/http"
	"time"

	"music-discovery/backend/usecase"

	"github.com/labstack/echo/v4"
)

func NewMetricsMiddleware(metrics usecase.IMetricsUsecase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			startedAt := time.Now()
			err := next(ctx)
			status := ctx.Response().Status
			if err != nil {
				var httpError *echo.HTTPError
				if errors.As(err, &httpError) {
					status = httpError.Code
				}
			}
			if err != nil && (status == 0 || status < 400) {
				status = http.StatusInternalServerError
			}
			if status == 0 {
				status = http.StatusOK
			}
			if metrics != nil && !isMonitoringPath(ctx.Request().URL.Path) {
				metrics.RecordRequest(ctx.Request().Context(), status, time.Since(startedAt))
			}
			return err
		}
	}
}

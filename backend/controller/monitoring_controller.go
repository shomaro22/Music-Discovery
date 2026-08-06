package controller

import (
	"errors"
	"net/http"
	"time"

	"music-discovery/backend/usecase"

	"github.com/labstack/echo/v4"
)

type IMonitoringController interface {
	Health(echo.Context) error
	Ready(echo.Context) error
	Metrics(echo.Context) error
	Alerts(echo.Context) error
	TestAlert(echo.Context) error
}

type MonitoringOption func(*monitoringController)

func WithHealthUsecase(value usecase.IHealthUsecase) MonitoringOption {
	return func(c *monitoringController) { c.health = value }
}
func WithAlertUsecase(value usecase.IAlertUsecase) MonitoringOption {
	return func(c *monitoringController) { c.alerts = value }
}
func WithAlertMonitor(value usecase.IAlertMonitor) MonitoringOption {
	return func(c *monitoringController) { c.alertMonitor = value }
}

type monitoringController struct {
	metrics      usecase.IMetricsUsecase
	health       usecase.IHealthUsecase
	alerts       usecase.IAlertUsecase
	alertMonitor usecase.IAlertMonitor
	version      string
	startedAt    time.Time
}

func NewMonitoringController(metrics usecase.IMetricsUsecase, version string, options ...MonitoringOption) IMonitoringController {
	c := &monitoringController{metrics: metrics, version: version, startedAt: time.Now()}
	for _, option := range options {
		option(c)
	}
	return c
}
func (c *monitoringController) Health(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, map[string]interface{}{"status": "ok", "version": c.version, "uptime_seconds": int64(time.Since(c.startedAt).Seconds())})
}
func (c *monitoringController) Ready(ctx echo.Context) error {
	if c.health == nil {
		return ctx.JSON(http.StatusServiceUnavailable, map[string]string{"status": "not_ready", "message": "health dependencies are not configured"})
	}
	result := c.health.Readiness(ctx.Request().Context())
	status := http.StatusOK
	if result.Status != "ready" {
		status = http.StatusServiceUnavailable
	}
	return ctx.JSON(status, result)
}
func (c *monitoringController) Metrics(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, c.metrics.Monitoring(ctx.Request().Context()))
}
func (c *monitoringController) Alerts(ctx echo.Context) error {
	if c.alertMonitor == nil {
		return ctx.JSON(http.StatusServiceUnavailable, map[string]string{"status": "unavailable", "message": "alert monitor is not configured"})
	}
	alerts, err := c.alertMonitor.ListCurrent(ctx.Request().Context())
	if err != nil {
		return writeAPIError(ctx, http.StatusInternalServerError, "ALERT_STATE_READ_FAILED", "Alert状態を取得できませんでした")
	}
	return ctx.JSON(http.StatusOK, map[string]interface{}{"alerts": alerts, "generated_at": time.Now().UTC()})
}
func (c *monitoringController) TestAlert(ctx echo.Context) error {
	if c.alertMonitor == nil {
		return ctx.JSON(http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
	}
	if err := c.alertMonitor.SendTest(ctx.Request().Context()); err != nil {
		if errors.Is(err, usecase.ErrAlertNotifierNotConfigured) {
			return writeAPIError(ctx, http.StatusServiceUnavailable, "ALERT_NOTIFIER_NOT_CONFIGURED", "Alert通知先が設定されていません")
		}
		return writeAPIError(ctx, http.StatusBadGateway, "ALERT_NOTIFICATION_FAILED", "Alert通知の送信に失敗しました")
	}
	return ctx.JSON(http.StatusOK, map[string]string{"status": "sent"})
}

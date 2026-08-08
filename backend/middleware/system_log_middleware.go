package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const (
	SystemLogStartedAtKey       = "system_log_started_at"
	SystemLogEventKey           = "system_log_event"
	SystemLogErrorCodeKey       = "system_log_error_code"
	SystemLogMessageKey         = "system_log_message"
	SystemLogExternalServiceKey = "system_log_external_service"
	SystemLogErrorDetailKey     = "system_log_error_detail"
	SystemLogCategoryKey        = "system_log_category"
	SystemLogFallbackUsedKey    = "system_log_fallback_used"
	SystemLogSpotifyErrorKey    = "system_log_spotify_error"
)

type SystemLogConfig struct {
	Environment   string
	Service       string
	Version       string
	Logger        *slog.Logger
	Metrics       usecase.IMetricsUsecase
	SaveTimeout   time.Duration
	SlowThreshold time.Duration
}

func NewSystemLogMiddleware(writer usecase.ISystemLogUsecase, configs ...SystemLogConfig) echo.MiddlewareFunc {
	config := SystemLogConfig{Environment: "local", Service: "music-discovery-api", Version: "development", SaveTimeout: 2 * time.Second}
	if len(configs) > 0 {
		config = configs[0]
	}
	if config.SaveTimeout <= 0 {
		config.SaveTimeout = 2 * time.Second
	}
	if config.SlowThreshold <= 0 {
		config.SlowThreshold = time.Second
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) (err error) {
			startedAt := time.Now()
			ctx.Set(SystemLogStartedAtKey, startedAt)
			defer func() {
				if recovered := recover(); recovered != nil {
					persistSystemLog(ctx, writer, config, http.StatusInternalServerError, "panic", "PANIC", "panic recovered", string(debug.Stack()))
					panic(recovered)
				}
			}()
			err = next(ctx)
			status := ctx.Response().Status
			if err != nil {
				var httpError *echo.HTTPError
				if errors.As(err, &httpError) {
					status = httpError.Code
				} else if status == 0 || status < 400 {
					status = http.StatusInternalServerError
				}
			}
			if status == 0 {
				status = http.StatusOK
			}
			if status < 400 && isMonitoringPath(ctx.Request().URL.Path) {
				return err
			}

			event, code, message := "request_completed", "", "request completed"
			if value := contextString(ctx, SystemLogEventKey); value != "" {
				event = value
			}
			if value := contextString(ctx, SystemLogErrorCodeKey); value != "" {
				code = value
			}
			if value := contextString(ctx, SystemLogMessageKey); value != "" {
				message = value
			}
			if event == "request_completed" && status >= 400 {
				switch status {
				case http.StatusUnauthorized:
					event = "authentication_failed"
					if code == "" {
						code = "UNAUTHORIZED"
					}
					if message == "" || message == "request completed" {
						message = "authentication failed"
					}
				case http.StatusForbidden:
					event = "authorization_denied"
					if code == "" {
						code = "FORBIDDEN"
					}
					if message == "" || message == "request completed" {
						message = "authorization denied"
					}
				case http.StatusTooManyRequests:
					event = "rate_limit_exceeded"
					if code == "" {
						code = "RATE_LIMITED"
					}
					if message == "" || message == "request completed" {
						message = "rate limit exceeded"
					}
				default:
					if status >= 500 {
						event = "server_error"
						if code == "" {
							code = "INTERNAL_ERROR"
						}
						if message == "" || message == "request completed" {
							message = "server error"
						}
					} else {
						event = "client_error"
						if code == "" {
							code = http.StatusText(status)
						}
						if message == "" || message == "request completed" {
							message = "client request failed"
						}
					}
				}
			}
			if status < 400 && event == "request_completed" && time.Since(startedAt) >= config.SlowThreshold {
				event, message = "slow_request", "request exceeded slow request threshold"
			}
			persistSystemLog(ctx, writer, config, status, event, code, message, "")
			return err
		}
	}
}

func persistSystemLog(ctx echo.Context, writer usecase.ISystemLogUsecase, config SystemLogConfig, status int, event, code, message, stack string) {
	if writer == nil {
		return
	}
	metadata, _ := json.Marshal(map[string]string{"remote_ip": ctx.RealIP(), "user_agent": ctx.Request().UserAgent()})
	var userID *uuid.UUID
	if value, ok := ctx.Get(AuthUserIDKey).(uuid.UUID); ok && value != uuid.Nil {
		userID = &value
	}
	requestID := ctx.Response().Header().Get(echo.HeaderXRequestID)
	if requestID == "" {
		requestID = ctx.Request().Header.Get(echo.HeaderXRequestID)
	}
	saveCtx, cancel := context.WithTimeout(context.Background(), config.SaveTimeout)
	defer cancel()
	err := writer.CreateSystemLog(saveCtx, &entity.SystemLog{
		ID: uuid.New(), Environment: config.Environment, Service: config.Service, Version: config.Version,
		Level: levelForEvent(status, event), Category: systemLogCategory(ctx, event), Event: event, ErrorCode: code,
		Message: message, RequestID: requestID, UserID: userID, Method: ctx.Request().Method, Path: ctx.Request().URL.Path,
		StatusCode: status, ExternalService: contextString(ctx, SystemLogExternalServiceKey), DurationMs: durationMilliseconds(ctx),
		ErrorDetail: contextString(ctx, SystemLogErrorDetailKey), Metadata: metadata, StackTrace: stack,
		IPAddress: ctx.RealIP(), UserAgent: ctx.Request().UserAgent(), CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		if config.Metrics != nil {
			config.Metrics.Increment(context.Background(), "system_log_save_failure")
		}
		if config.Logger != nil {
			config.Logger.Error("system log persistence failed", "error", err, "request_id", requestID, "event", event)
		}
	}
	if config.Metrics != nil {
		recordEventMetric(config.Metrics, ctx, event)
		if used, _ := ctx.Get(SystemLogFallbackUsedKey).(bool); used {
			config.Metrics.Increment(context.Background(), "recommendation_fallback")
		}
	}
}

func isMonitoringPath(path string) bool {
	return path == "/health" || path == "/ready" || path == "/api/system/metrics" || path == "/api/system/alerts"
}
func recordEventMetric(metrics usecase.IMetricsUsecase, ctx echo.Context, event string) {
	switch event {
	case "login_succeeded":
		metrics.Increment(context.Background(), "login_success")
	case "login_failed":
		metrics.Increment(context.Background(), "login_failure")
	case "recommendation_succeeded":
		metrics.Increment(context.Background(), "recommendation_success")
	case "recommendation_failed":
		metrics.Increment(context.Background(), "recommendation_failure")
	case "recommendation_fallback":
		metrics.Increment(context.Background(), "recommendation_fallback")
	case "feedback_saved":
		metrics.Increment(context.Background(), "feedback_saved")
	case "daily_discovery_generated":
		metrics.Increment(context.Background(), "daily_discovery_success")
	case "rate_limit_exceeded":
		metrics.Increment(context.Background(), "rate_limit_exceeded")
	}
	if spotifyError, _ := ctx.Get(SystemLogSpotifyErrorKey).(bool); spotifyError {
		metrics.Increment(context.Background(), "spotify_api_error")
	}
}

func levelForEvent(status int, event string) string {
	if event == "slow_request" {
		return "warn"
	}
	if status >= 500 {
		return "error"
	}
	if status >= 400 {
		return "warn"
	}
	return "info"
}

func contextString(ctx echo.Context, key string) string {
	value, _ := ctx.Get(key).(string)
	return value
}
func durationMilliseconds(ctx echo.Context) int64 {
	startedAt, ok := ctx.Get(SystemLogStartedAtKey).(time.Time)
	if !ok {
		return 0
	}
	return time.Since(startedAt).Milliseconds()
}
func systemLogCategoryForEvent(event string) string {
	switch event {
	case "database_connection_failed", "database_query_failed", "database_transaction_failed":
		return "database"
	default:
		return ""
	}
}

func systemLogCategory(ctx echo.Context, event string) string {
	if value := contextString(ctx, SystemLogCategoryKey); value != "" {
		return value
	}
	switch event {
	case "request_completed":
		return "access"
	case "authentication_failed", "authorization_denied", "rate_limit_exceeded", "suspended_user_login_attempt", "access_token_invalid", "token_version_mismatch":
		return "security"
	case "database_connection_failed":
		return "database"
	case "redis_connection_failed":
		return "redis"
	case "spotify_api_failed", "spotify_api_rate_limited":
		return "external_service"
	case "recommendation_failed", "daily_discovery_failed":
		return "recommendation"
	case "transaction_rollback":
		return "transaction"
	case "job_completed", "job_failed":
		return "job"
	default:
		return "application"
	}
}

package middleware

import (
	"encoding/json"
	"errors"
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
)

func NewSystemLogMiddleware(writer usecase.ISystemLogUsecase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) (err error) {
			startedAt := time.Now()
			ctx.Set(SystemLogStartedAtKey, startedAt)
			defer func() {
				if recovered := recover(); recovered != nil {
					stack := string(debug.Stack())
					persistSystemLog(ctx, writer, http.StatusInternalServerError, "panic", "PANIC", "panic recovered", stack)
					panic(recovered)
				}
			}()
			err = next(ctx)
			status := ctx.Response().Status
			code := ""
			event := "request_error"
			message := "request failed"
			if value, ok := ctx.Get(SystemLogEventKey).(string); ok && value != "" {
				event = value
			}
			if value, ok := ctx.Get(SystemLogErrorCodeKey).(string); ok && value != "" {
				code = value
			}
			if value, ok := ctx.Get(SystemLogMessageKey).(string); ok && value != "" {
				message = value
			}
			if err != nil {
				var httpError *echo.HTTPError
				if errors.As(err, &httpError) {
					status = httpError.Code
				}
			}
			if status == 0 {
				status = http.StatusOK
			}
			if status < 400 {
				return err
			}
			if event == "request_error" {
				switch status {
				case http.StatusUnauthorized:
					event, code, message = "authentication_failed", "UNAUTHORIZED", "authentication failed"
				case http.StatusForbidden:
					event, code, message = "authorization_denied", "FORBIDDEN", "authorization denied"
				case http.StatusTooManyRequests:
					event, code, message = "rate_limit_exceeded", "RATE_LIMITED", "rate limit exceeded"
				default:
					if status >= 500 {
						event, code, message = "server_error", "INTERNAL_ERROR", "server error"
					} else {
						event, code, message = "client_error", http.StatusText(status), "client request failed"
					}
				}
			}
			persistSystemLog(ctx, writer, status, event, code, message, "")
			return err
		}
	}
}

func persistSystemLog(ctx echo.Context, writer usecase.ISystemLogUsecase, status int, event, code, message, stack string) {
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
	_ = writer.CreateSystemLog(ctx.Request().Context(), &entity.SystemLog{ID: uuid.New(), Level: levelForStatus(status), Category: systemLogCategory(ctx, event), Event: event, ErrorCode: code, Message: message, RequestID: requestID, UserID: userID, Method: ctx.Request().Method, Path: ctx.Request().URL.Path, StatusCode: status, ExternalService: contextString(ctx, SystemLogExternalServiceKey), DurationMs: durationMilliseconds(ctx), ErrorDetail: contextString(ctx, SystemLogErrorDetailKey), Metadata: metadata, StackTrace: stack, IPAddress: ctx.RealIP(), UserAgent: ctx.Request().UserAgent(), CreatedAt: time.Now().UTC()})
}
func levelForStatus(status int) string {
	if status >= 500 {
		return "error"
	}
	return "warn"
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

func systemLogCategory(ctx echo.Context, event string) string {
	if value := contextString(ctx, SystemLogCategoryKey); value != "" {
		return value
	}
	switch event {
	case "authentication_failed", "suspended_user_login_attempt", "access_token_invalid", "token_version_mismatch":
		return "auth"
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

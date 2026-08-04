package middleware

import (
	"errors"
	"log/slog"
	"net/http"

	"music-discovery/backend/entity"

	"github.com/labstack/echo/v4"
)

func NewHTTPErrorHandler(logger *slog.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		status := http.StatusInternalServerError
		code := "INTERNAL_ERROR"
		message := "サーバー内部でエラーが発生しました"

		var httpError *echo.HTTPError
		if errors.As(err, &httpError) {
			status = httpError.Code
			if errors.Is(httpError.Internal, entity.ErrUserSuspended) {
				code = "USER_SUSPENDED"
				message = "このアカウントは現在利用できません"
			} else if messageError, ok := httpError.Message.(error); ok && errors.Is(messageError, entity.ErrUserSuspended) {
				code = "USER_SUSPENDED"
				message = "このアカウントは現在利用できません"
			} else {
				switch status {
				case http.StatusUnauthorized:
					code = "UNAUTHORIZED"
					message = "認証情報が無効または期限切れです"
				case http.StatusForbidden:
					code = "FORBIDDEN"
					message = "この操作を実行する権限がありません"
				case http.StatusBadRequest:
					code = "BAD_REQUEST"
					message = "リクエスト内容が不正です"
				case http.StatusNotFound:
					code = "NOT_FOUND"
					message = "対象のリソースが見つかりません"
				case http.StatusMethodNotAllowed:
					code = "METHOD_NOT_ALLOWED"
					message = "この操作はサポートされていません"
				case http.StatusRequestEntityTooLarge:
					code = "PAYLOAD_TOO_LARGE"
					message = "リクエストサイズが上限を超えています"
				case http.StatusUnsupportedMediaType:
					code = "UNSUPPORTED_MEDIA_TYPE"
					message = "リクエストはJSON形式で送信してください"
				case http.StatusTooManyRequests:
					code = "RATE_LIMIT_EXCEEDED"
					message = "しばらく時間を置いてから再度お試しください"
				}
			}
		}

		requestID := c.Response().Header().Get(echo.HeaderXRequestID)
		if requestID == "" {
			requestID = c.Request().Header.Get(echo.HeaderXRequestID)
		}

		if status >= http.StatusInternalServerError {
			logger.Error("request failed",
				"error", err,
				"method", c.Request().Method,
				"path", c.Request().URL.Path,
				"request_id", requestID,
			)
		}

		if writeErr := c.JSON(status, entity.APIErrorResponse{
			Status: status, Code: code, Message: message, RequestID: requestID,
		}); writeErr != nil {
			logger.Error("failed to write error response", "error", writeErr, "request_id", requestID)
		}
	}
}

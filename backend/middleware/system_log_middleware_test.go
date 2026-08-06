package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"music-discovery/backend/entity"
)

type systemLogUsecaseStub struct{ logs []entity.SystemLog }

func (s *systemLogUsecaseStub) CreateSystemLog(_ context.Context, value *entity.SystemLog) error {
	s.logs = append(s.logs, *value)
	return nil
}
func TestSystemLogMiddlewareStoresAuthorizationFailure(t *testing.T) {
	e := echo.New()
	writer := &systemLogUsecaseStub{}
	handler := NewSystemLogMiddleware(writer)(func(echo.Context) error { return echo.NewHTTPError(http.StatusForbidden) })
	request := httptest.NewRequest(http.MethodGet, "/api/system/users", nil)
	recorder := httptest.NewRecorder()
	ctx := e.NewContext(request, recorder)
	_ = handler(ctx)
	if len(writer.logs) != 1 {
		t.Fatalf("expected one log, got %d", len(writer.logs))
	}
	if writer.logs[0].Event != "authorization_denied" || writer.logs[0].StatusCode != http.StatusForbidden {
		t.Fatalf("unexpected log: %+v", writer.logs[0])
	}
}

func TestSystemLogMiddlewarePreservesApplicationErrorCode(t *testing.T) {
	e := echo.New()
	writer := &systemLogUsecaseStub{}
	handler := NewSystemLogMiddleware(writer)(func(ctx echo.Context) error {
		ctx.Set(SystemLogErrorCodeKey, "RECOMMENDATION_SAVE_FAILED")
		ctx.Set(SystemLogMessageKey, "save failed")
		return ctx.JSON(http.StatusInternalServerError, map[string]string{"status": "error"})
	})
	ctx := e.NewContext(httptest.NewRequest(http.MethodPost, "/api/recommendations", nil), httptest.NewRecorder())
	_ = handler(ctx)
	if got := writer.logs[0].ErrorCode; got != "RECOMMENDATION_SAVE_FAILED" {
		t.Fatalf("code=%s", got)
	}
}

func TestSystemLogMiddlewareRecordsSlowRequestAsWarn(t *testing.T) {
	e := echo.New()
	writer := &systemLogUsecaseStub{}
	handler := NewSystemLogMiddleware(writer, SystemLogConfig{SlowThreshold: time.Nanosecond})(func(ctx echo.Context) error { return ctx.NoContent(http.StatusOK) })
	ctx := e.NewContext(httptest.NewRequest(http.MethodGet, "/api/test", nil), httptest.NewRecorder())
	_ = handler(ctx)
	if writer.logs[0].Event != "slow_request" || writer.logs[0].Level != "warn" {
		t.Fatalf("log=%+v", writer.logs[0])
	}
}

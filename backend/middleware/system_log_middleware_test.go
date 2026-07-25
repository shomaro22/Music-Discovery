package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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

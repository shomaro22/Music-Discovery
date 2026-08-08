package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"music-discovery/backend/entity"
	"music-discovery/backend/middleware"
	"music-discovery/backend/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type roleAdminUsecaseStub struct {
	usecase.ISystemAdminUsecase
	called bool
}

func (s *roleAdminUsecaseStub) ChangeRole(_ context.Context, actorID, userID uuid.UUID, input usecase.RoleChangeInput, _ usecase.AuditInput) (*usecase.RoleChangeResponse, error) {
	s.called = true
	return &usecase.RoleChangeResponse{UserID: userID, BeforeRole: entity.RoleUser, AfterRole: input.Role}, nil
}

func TestRoleAdminControllerChangeRoleReturnsNoContent(t *testing.T) {
	e := echo.New()
	uc := &roleAdminUsecaseStub{}
	controller := NewRoleAdminController(uc)
	userID := uuid.New()
	actorID := uuid.New()
	req := httptest.NewRequest(http.MethodPatch, "/api/system/users/"+userID.String()+"/role", strings.NewReader(`{"role":"content_admin","reason":"promotion"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.SetPath("/api/system/users/:userId/role")
	ctx.SetParamNames("userId")
	ctx.SetParamValues(userID.String())
	ctx.Set(middleware.AuthUserIDKey, actorID)

	if err := controller.ChangeRole(ctx); err != nil {
		t.Fatal(err)
	}
	if !uc.called {
		t.Fatal("ChangeRole usecase was not called")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d want=%d body=%s", rec.Code, http.StatusNoContent, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty response body, got %q", rec.Body.String())
	}
}

package controller

import (
	"errors"
	"net/http"

	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type ChangeRoleRequest struct {
	Role entity.Role `json:"role"`
}

type IAdminController interface {
	ProtectedPlaceholder(echo.Context) error
	ChangeRole(echo.Context) error
}

type adminController struct{ adminUsecase usecase.IAdminUsecase }

func NewAdminController(adminUsecase usecase.IAdminUsecase) IAdminController {
	return &adminController{adminUsecase: adminUsecase}
}
func (c *adminController) ProtectedPlaceholder(ctx echo.Context) error {
	return ctx.NoContent(http.StatusNoContent)
}

func (c *adminController) ChangeRole(ctx echo.Context) error {
	actorID, ok := ctx.Get("auth_user_id").(uuid.UUID)
	if !ok || actorID == uuid.Nil {
		return writeAPIError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です")
	}
	targetID, err := uuid.Parse(ctx.Param("userId"))
	if err != nil {
		return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "入力内容を確認してください")
	}
	var request ChangeRoleRequest
	if err := ctx.Bind(&request); err != nil || !request.Role.IsPersistable() {
		return writeAPIError(ctx, http.StatusBadRequest, "INVALID_ROLE", "Roleを確認してください")
	}
	err = c.adminUsecase.ChangeRole(ctx.Request().Context(), usecase.ChangeRoleInput{ActorUserID: actorID, TargetUserID: targetID, Role: request.Role})
	switch {
	case err == nil:
		return ctx.NoContent(http.StatusNoContent)
	case errors.Is(err, entity.ErrForbidden):
		return writeAPIError(ctx, http.StatusForbidden, "FORBIDDEN", "権限がありません")
	case errors.Is(err, entity.ErrUserNotFound):
		return writeAPIError(ctx, http.StatusNotFound, "USER_NOT_FOUND", "Userが見つかりません")
	case errors.Is(err, entity.ErrInvalidInput):
		return writeAPIError(ctx, http.StatusBadRequest, "INVALID_ROLE", "Roleを確認してください")
	default:
		return writeAPIError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "サーバー内部でエラーが発生しました")
	}
}

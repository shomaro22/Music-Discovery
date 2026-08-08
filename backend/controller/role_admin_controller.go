package controller

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"
)

type IRoleAdminController interface {
	ListRoles(echo.Context) error
	ChangeRole(echo.Context) error
}

type roleAdminController struct {
	usecase usecase.ISystemAdminUsecase
}

type roleChangeAdminRequest struct {
	Role   entity.Role `json:"role"`
	Reason string      `json:"reason"`
}

type roleListResponse struct {
	Roles []entity.Role `json:"roles"`
}

func NewRoleAdminController(value usecase.ISystemAdminUsecase) IRoleAdminController {
	return &roleAdminController{usecase: value}
}

func (c *roleAdminController) ListRoles(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, roleListResponse{Roles: c.usecase.ListRoles(ctx.Request().Context())})
}

func (c *roleAdminController) ChangeRole(ctx echo.Context) error {
	actorID, err := adminActorID(ctx)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	id, err := adminPathID(ctx, "userId")
	if err != nil {
		return adminResponseError(ctx, entity.ErrInvalidInput)
	}
	var request roleChangeAdminRequest
	if err := ctx.Bind(&request); err != nil {
		return adminResponseError(ctx, entity.ErrInvalidInput)
	}
	_, err = c.usecase.ChangeRole(ctx.Request().Context(), actorID, id, usecase.RoleChangeInput{
		Role:   request.Role,
		Reason: request.Reason,
	}, adminAuditInput(ctx))
	if err != nil {
		return adminResponseError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

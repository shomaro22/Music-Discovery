package controller

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"
)

type IUserAdminController interface {
	ListUsers(echo.Context) error
	GetUser(echo.Context) error
	SetUserStatus(echo.Context) error
}

type userAdminController struct {
	usecase usecase.ISystemAdminUsecase
}

type userStatusAdminRequest struct {
	Status entity.UserStatus `json:"status"`
	Reason string            `json:"reason"`
}

func NewUserAdminController(value usecase.ISystemAdminUsecase) IUserAdminController {
	return &userAdminController{usecase: value}
}

func (c *userAdminController) ListUsers(ctx echo.Context) error {
	input, err := adminPageInput(ctx)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	value, err := c.usecase.ListUsers(ctx.Request().Context(), input)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, mapAdminPage(value, toUserAdminResponse))
}

func (c *userAdminController) GetUser(ctx echo.Context) error {
	id, err := adminPathID(ctx, "userId")
	if err != nil {
		return adminResponseError(ctx, entity.ErrInvalidInput)
	}
	value, err := c.usecase.GetUser(ctx.Request().Context(), id)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, toUserDetailAdminResponse(*value))
}

func (c *userAdminController) SetUserStatus(ctx echo.Context) error {
	actorID, err := adminActorID(ctx)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	id, err := adminPathID(ctx, "userId")
	if err != nil {
		return adminResponseError(ctx, entity.ErrInvalidInput)
	}
	var request userStatusAdminRequest
	if err := ctx.Bind(&request); err != nil {
		return adminResponseError(ctx, entity.ErrInvalidInput)
	}
	if err := c.usecase.SetUserStatus(ctx.Request().Context(), actorID, id, usecase.UserStatusInput{
		Status: request.Status,
		Reason: request.Reason,
	}, adminAuditInput(ctx)); err != nil {
		return adminResponseError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

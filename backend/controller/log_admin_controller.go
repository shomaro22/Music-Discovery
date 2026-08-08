package controller

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"
)

type ILogAdminController interface {
	ListAuditLogs(echo.Context) error
	GetAuditLog(echo.Context) error
	ListSystemLogs(echo.Context) error
	GetSystemLog(echo.Context) error
}

type logAdminController struct {
	usecase usecase.ISystemAdminUsecase
}

func NewLogAdminController(value usecase.ISystemAdminUsecase) ILogAdminController {
	return &logAdminController{usecase: value}
}

func (c *logAdminController) ListAuditLogs(ctx echo.Context) error {
	input, err := adminPageInput(ctx)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	value, err := c.usecase.ListAuditLogs(ctx.Request().Context(), input)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, mapAdminPage(value, toAuditLogListAdminResponse))
}

func (c *logAdminController) GetAuditLog(ctx echo.Context) error {
	id, err := adminPathID(ctx, "auditLogId")
	if err != nil {
		return adminResponseError(ctx, entity.ErrInvalidInput)
	}
	value, err := c.usecase.GetAuditLog(ctx.Request().Context(), id)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, value)
}

func (c *logAdminController) ListSystemLogs(ctx echo.Context) error {
	input, err := adminPageInput(ctx)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	value, err := c.usecase.ListSystemLogs(ctx.Request().Context(), input)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, mapAdminPage(value, toSystemLogListAdminResponse))
}

func (c *logAdminController) GetSystemLog(ctx echo.Context) error {
	id, err := adminPathID(ctx, "logId")
	if err != nil {
		return adminResponseError(ctx, entity.ErrInvalidInput)
	}
	value, err := c.usecase.GetSystemLog(ctx.Request().Context(), id)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, value)
}

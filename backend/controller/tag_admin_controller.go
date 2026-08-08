package controller

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"
)

type ITagAdminController interface {
	ListTags(echo.Context) error
	ListTagOptions(echo.Context) error
	GetTag(echo.Context) error
	CreateTag(echo.Context) error
	UpdateTag(echo.Context) error
	DeleteTag(echo.Context) error
}

type tagAdminController struct {
	usecase usecase.IContentAdminUsecase
}

type tagAdminRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

func NewTagAdminController(value usecase.IContentAdminUsecase) ITagAdminController {
	return &tagAdminController{usecase: value}
}

func (c *tagAdminController) ListTags(ctx echo.Context) error {
	input, err := adminPageInput(ctx)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	value, err := c.usecase.ListTags(ctx.Request().Context(), input)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, mapAdminPage(value, toTagAdminResponse))
}

func (c *tagAdminController) ListTagOptions(ctx echo.Context) error {
	value, err := c.usecase.ListTagOptions(ctx.Request().Context())
	if err != nil {
		return adminResponseError(ctx, err)
	}
	items := make([]tagOptionAdminResponse, 0, len(value))
	for _, item := range value {
		items = append(items, toTagOptionAdminResponse(item))
	}
	return ctx.JSON(http.StatusOK, items)
}

func (c *tagAdminController) GetTag(ctx echo.Context) error {
	id, err := adminPathID(ctx, "tagId")
	if err != nil {
		return adminResponseError(ctx, entity.ErrInvalidInput)
	}
	value, err := c.usecase.GetTag(ctx.Request().Context(), id)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, toTagAdminResponse(*value))
}

func (c *tagAdminController) CreateTag(ctx echo.Context) error {
	return c.saveTag(ctx, uuid.Nil, http.StatusCreated)
}

func (c *tagAdminController) UpdateTag(ctx echo.Context) error {
	id, err := adminPathID(ctx, "tagId")
	if err != nil {
		return adminResponseError(ctx, entity.ErrInvalidInput)
	}
	return c.saveTag(ctx, id, http.StatusOK)
}

func (c *tagAdminController) saveTag(ctx echo.Context, id uuid.UUID, status int) error {
	actorID, err := adminActorID(ctx)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	var request tagAdminRequest
	if err := ctx.Bind(&request); err != nil {
		return adminResponseError(ctx, entity.ErrInvalidInput)
	}
	value, err := c.usecase.SaveTag(ctx.Request().Context(), actorID, &entity.Tag{
		ID:          id,
		Name:        request.Name,
		Slug:        request.Slug,
		Category:    request.Category,
		Description: request.Description,
	}, adminAuditInput(ctx))
	if err != nil {
		return adminResponseError(ctx, err)
	}
	return ctx.JSON(status, value)
}

func (c *tagAdminController) DeleteTag(ctx echo.Context) error {
	actorID, err := adminActorID(ctx)
	if err != nil {
		return adminResponseError(ctx, err)
	}
	id, err := adminPathID(ctx, "tagId")
	if err != nil {
		return adminResponseError(ctx, entity.ErrInvalidInput)
	}
	if err := c.usecase.DeleteTag(ctx.Request().Context(), actorID, id, adminAuditInput(ctx)); err != nil {
		return adminResponseError(ctx, err)
	}
	return ctx.NoContent(http.StatusNoContent)
}

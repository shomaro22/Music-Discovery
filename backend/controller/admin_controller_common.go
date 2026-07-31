package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func adminActorID(ctx echo.Context) (uuid.UUID, error) {
	value, ok := ctx.Get("auth_user_id").(uuid.UUID)
	switch {
	case !ok, value == uuid.Nil:
		return uuid.Nil, entity.ErrInvalidCredentials
	default:
		return value, nil
	}
}

func adminPathID(ctx echo.Context, name string) (uuid.UUID, error) {
	return uuid.Parse(ctx.Param(name))
}

func adminAuditInput(ctx echo.Context) usecase.AuditInput {
	return usecase.AuditInput{
		IPAddress: ctx.RealIP(),
		UserAgent: ctx.Request().UserAgent(),
		RequestID: ctx.Response().Header().Get(echo.HeaderXRequestID),
	}
}

func queryInt(ctx echo.Context, name string, defaultValue int) (int, error) {
	raw := strings.TrimSpace(ctx.QueryParam(name))
	switch raw {
	case "":
		return defaultValue, nil
	default:
		value, err := strconv.Atoi(raw)
		if err != nil {
			return 0, entity.ErrInvalidInput
		}
		return value, nil
	}
}

func queryOptionalInt(ctx echo.Context, name string) (*int, error) {
	raw := strings.TrimSpace(ctx.QueryParam(name))
	switch raw {
	case "":
		return nil, nil
	default:
		value, err := strconv.Atoi(raw)
		if err != nil {
			return nil, entity.ErrInvalidInput
		}
		return &value, nil
	}
}

func queryOptionalBool(ctx echo.Context, name string) (*bool, error) {
	raw := strings.TrimSpace(ctx.QueryParam(name))
	switch raw {
	case "":
		return nil, nil
	default:
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, entity.ErrInvalidInput
		}
		return &value, nil
	}
}

func queryBool(ctx echo.Context, name string, defaultValue bool) (bool, error) {
	value, err := queryOptionalBool(ctx, name)
	switch {
	case err != nil:
		return false, err
	case value == nil:
		return defaultValue, nil
	default:
		return *value, nil
	}
}

func adminPageInput(ctx echo.Context) (usecase.PageInput, error) {
	page, err := queryInt(ctx, "page", 1)
	if err != nil {
		return usecase.PageInput{}, err
	}
	limit, err := queryInt(ctx, "limit", 20)
	if err != nil {
		return usecase.PageInput{}, err
	}
	explicit, err := queryOptionalBool(ctx, "explicit")
	if err != nil {
		return usecase.PageInput{}, err
	}
	statusCode, err := queryOptionalInt(ctx, "status_code")
	if err != nil {
		return usecase.PageInput{}, err
	}
	includeDeleted, err := queryBool(ctx, "include_deleted", false)
	if err != nil {
		return usecase.PageInput{}, err
	}

	return usecase.PageInput{
		Keyword:         ctx.QueryParam("keyword"),
		Sort:            ctx.QueryParam("sort"),
		Order:           ctx.QueryParam("order"),
		Page:            page,
		Limit:           limit,
		TagID:           ctx.QueryParam("tag_id"),
		ArtistID:        ctx.QueryParam("artist_id"),
		Explicit:        explicit,
		Role:            strings.TrimSpace(ctx.QueryParam("role")),
		Status:          strings.TrimSpace(ctx.QueryParam("status")),
		IncludeDeleted:  includeDeleted,
		Action:          ctx.QueryParam("action"),
		ActorID:         ctx.QueryParam("actor_id"),
		TargetType:      ctx.QueryParam("target_type"),
		TargetID:        ctx.QueryParam("target_id"),
		RequestID:       ctx.QueryParam("request_id"),
		Level:           ctx.QueryParam("level"),
		Category:        ctx.QueryParam("category"),
		Event:           ctx.QueryParam("event"),
		UserID:          ctx.QueryParam("user_id"),
		ExternalService: ctx.QueryParam("external_service"),
		Method:          ctx.QueryParam("method"),
		Path:            ctx.QueryParam("path"),
		TagCategory:     ctx.QueryParam("tag_category"),
		StatusCode:      statusCode,
		From:            ctx.QueryParam("from"),
		To:              ctx.QueryParam("to"),
	}, nil
}

type adminErrorResponse struct {
	status  int
	code    string
	message string
}

func resolveAdminError(err error) adminErrorResponse {
	switch {
	case errors.Is(err, entity.ErrInvalidCredentials):
		return adminErrorResponse{http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です"}
	case errors.Is(err, entity.ErrInvalidInput), errors.Is(err, entity.ErrReasonRequired):
		return adminErrorResponse{http.StatusBadRequest, "INVALID_INPUT", "入力内容を確認してください"}
	case errors.Is(err, entity.ErrNotFound), errors.Is(err, entity.ErrUserNotFound):
		return adminErrorResponse{http.StatusNotFound, "NOT_FOUND", "対象が見つかりません"}
	case errors.Is(err, entity.ErrConflict), errors.Is(err, entity.ErrAlreadyActive), errors.Is(err, entity.ErrAlreadySuspended), errors.Is(err, entity.ErrLastSystemAdmin):
		return adminErrorResponse{http.StatusConflict, "CONFLICT", "現在の状態では操作できません"}
	case errors.Is(err, entity.ErrForbidden):
		return adminErrorResponse{http.StatusForbidden, "FORBIDDEN", "権限がありません"}
	case errors.Is(err, entity.ErrSpotifyUnavailable):
		return adminErrorResponse{http.StatusServiceUnavailable, "SPOTIFY_UNAVAILABLE", "Spotifyを現在利用できません"}
	default:
		return adminErrorResponse{http.StatusInternalServerError, "INTERNAL_ERROR", "サーバー内部でエラーが発生しました"}
	}
}

func adminResponseError(ctx echo.Context, err error) error {
	response := resolveAdminError(err)
	return writeAPIError(ctx, response.status, response.code, response.message)
}

func bindAdminRequest(ctx echo.Context, request interface{}) error {
	if err := ctx.Bind(request); err != nil {
		return entity.ErrInvalidInput
	}
	return nil
}

package controller

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"
)

type IRecommendationHistoryController interface {
	List(echo.Context) error
	GetByID(echo.Context) error
}
type recommendationHistoryController struct {
	usecase usecase.IRecommendationHistoryUsecase
}

func NewRecommendationHistoryController(value usecase.IRecommendationHistoryUsecase) IRecommendationHistoryController {
	return &recommendationHistoryController{usecase: value}
}

type RecommendationHistorySongResponse struct {
	RecommendationDetailID uuid.UUID                 `json:"recommendation_detail_id"`
	RecommendationType     entity.RecommendationType `json:"recommendation_type"`
	DisplayOrder           int                       `json:"display_order"`
	SongID                 uuid.UUID                 `json:"song_id"`
	SongName               string                    `json:"song_name"`
	ArtistID               uuid.UUID                 `json:"artist_id"`
	ArtistName             string                    `json:"artist_name"`
	ArtworkURL             string                    `json:"artwork_url"`
	Reason                 string                    `json:"reason"`
	FeedbackStatus         *entity.FeedbackStatus    `json:"feedback_status"`
}
type RecommendationHistoryItemResponse struct {
	RecommendationID uuid.UUID                           `json:"recommendation_id"`
	Source           entity.RecommendationSource         `json:"source"`
	TargetDate       *string                             `json:"target_date,omitempty"`
	RecommendedAt    time.Time                           `json:"recommended_at"`
	Songs            []RecommendationHistorySongResponse `json:"songs"`
}
type RecommendationHistoryListResponse struct {
	Items      []RecommendationHistoryItemResponse `json:"items"`
	Page       int                                 `json:"page"`
	Limit      int                                 `json:"limit"`
	Total      int64                               `json:"total"`
	TotalPages int                                 `json:"total_pages"`
	Timezone   string                              `json:"timezone"`
}
type RecommendationHistoryDetailResponse struct {
	RecommendationID uuid.UUID                    `json:"recommendation_id"`
	Source           entity.RecommendationSource  `json:"source"`
	TargetDate       *string                      `json:"target_date,omitempty"`
	RecommendedAt    time.Time                    `json:"recommended_at"`
	Recommendations  []RecommendationItemResponse `json:"recommendations"`
	Timezone         string                       `json:"timezone"`
}

func historyItem(value entity.Recommendation) RecommendationHistoryItemResponse {
	var target *string
	if value.TargetDate != nil {
		formatted := value.TargetDate.Format("2006-01-02")
		target = &formatted
	}
	songs := make([]RecommendationHistorySongResponse, 0, len(value.Details))
	for _, detail := range value.Details {
		songs = append(songs, RecommendationHistorySongResponse{
			RecommendationDetailID: detail.ID,
			RecommendationType:     detail.Type,
			DisplayOrder:           detail.Position,
			SongID:                 detail.SongID,
			SongName:               detail.Song.Name,
			ArtistID:               detail.Song.ArtistID,
			ArtistName:             detail.Song.Artist.Name,
			ArtworkURL:             detail.Song.ImageURL,
			Reason:                 detail.Reason,
			FeedbackStatus:         detail.FeedbackStatus,
		})
	}
	return RecommendationHistoryItemResponse{value.ID, value.Source, target, value.RecommendedAt, songs}
}
func historyDetail(value *entity.Recommendation) RecommendationHistoryDetailResponse {
	base := toRecommendationResponse(value)
	var target *string
	if value.TargetDate != nil {
		formatted := value.TargetDate.Format("2006-01-02")
		target = &formatted
	}
	return RecommendationHistoryDetailResponse{RecommendationID: value.ID, Source: value.Source, TargetDate: target, RecommendedAt: value.RecommendedAt, Recommendations: base.Recommendations}
}
func parsePositive(value string, fallback int) (int, error) {
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}
func (c *recommendationHistoryController) List(ctx echo.Context) error {
	userID, ok := authUserID(ctx)
	if !ok {
		return writeAPIError(ctx, 401, "UNAUTHORIZED", "認証が必要です")
	}
	page, err := parsePositive(ctx.QueryParam("page"), 1)
	if err != nil {
		return c.handle(ctx, entity.ErrInvalidPage)
	}
	limit, err := parsePositive(ctx.QueryParam("limit"), usecase.RecommendationHistoryDefaultLimit)
	if err != nil {
		return c.handle(ctx, entity.ErrInvalidLimit)
	}
	var source *entity.RecommendationSource
	if raw := ctx.QueryParam("source"); raw != "" {
		parsed := entity.RecommendationSource(raw)
		source = &parsed
	}
	result, err := c.usecase.List(ctx.Request().Context(), userID, usecase.RecommendationHistoryInput{Page: page, Limit: limit, Source: source, StartDate: ctx.QueryParam("startDate"), EndDate: ctx.QueryParam("endDate")})
	if err != nil {
		return c.handle(ctx, err)
	}
	items := make([]RecommendationHistoryItemResponse, 0, len(result.Items))
	for _, v := range result.Items {
		items = append(items, historyItem(v))
	}
	return ctx.JSON(http.StatusOK, RecommendationHistoryListResponse{Items: items, Page: result.Page, Limit: result.Limit, Total: result.Total, TotalPages: result.TotalPages, Timezone: result.Timezone})
}
func (c *recommendationHistoryController) GetByID(ctx echo.Context) error {
	userID, ok := authUserID(ctx)
	if !ok {
		return writeAPIError(ctx, 401, "UNAUTHORIZED", "認証が必要です")
	}
	id, err := uuid.Parse(ctx.Param("recommendationId"))
	if err != nil {
		return c.handle(ctx, entity.ErrInvalidRecommendationID)
	}
	value, timezone, err := c.usecase.GetByID(ctx.Request().Context(), userID, id)
	if err != nil {
		return c.handle(ctx, err)
	}
	response := historyDetail(value)
	response.Timezone = timezone
	return ctx.JSON(http.StatusOK, response)
}
func (c *recommendationHistoryController) handle(ctx echo.Context, err error) error {
	switch {
	case errors.Is(err, entity.ErrInvalidRecommendationID):
		return writeAPIError(ctx, 400, "INVALID_RECOMMENDATION_ID", "Recommendation IDが不正です")
	case errors.Is(err, entity.ErrInvalidPage):
		return writeAPIError(ctx, 400, "INVALID_PAGE", "Page指定が不正です")
	case errors.Is(err, entity.ErrInvalidLimit):
		return writeAPIError(ctx, 400, "INVALID_LIMIT", "Limit指定が不正です")
	case errors.Is(err, entity.ErrInvalidRecommendationSource):
		return writeAPIError(ctx, 400, "INVALID_RECOMMENDATION_SOURCE", "Source指定が不正です")
	case errors.Is(err, entity.ErrInvalidDateRange):
		return writeAPIError(ctx, 400, "INVALID_DATE_RANGE", "日付範囲が不正です")
	case errors.Is(err, entity.ErrUserSuspended):
		return writeAPIError(ctx, 403, "USER_SUSPENDED", "利用停止中です")
	case errors.Is(err, entity.ErrRecommendationHistoryAccessDenied):
		return writeAPIError(ctx, 403, "RECOMMENDATION_HISTORY_ACCESS_DENIED", "この履歴へアクセスできません")
	case errors.Is(err, entity.ErrRecommendationHistoryNotFound):
		return writeAPIError(ctx, 404, "RECOMMENDATION_HISTORY_NOT_FOUND", "推薦履歴が見つかりません")
	default:
		return writeAPIError(ctx, 500, "RECOMMENDATION_HISTORY_FETCH_FAILED", "推薦履歴を取得できませんでした")
	}
}

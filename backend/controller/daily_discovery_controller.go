package controller

import (
	"errors"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"
	"net/http"
	"strconv"
	"time"
)

type IDailyDiscoveryController interface {
	GetToday(echo.Context) error
	List(echo.Context) error
	GetByID(echo.Context) error
}
type dailyDiscoveryController struct {
	usecase usecase.IDailyDiscoveryUsecase
}

func NewDailyDiscoveryController(u usecase.IDailyDiscoveryUsecase) IDailyDiscoveryController {
	return &dailyDiscoveryController{u}
}

type DailyDiscoveryResponse struct {
	RecommendationID uuid.UUID                    `json:"recommendation_id"`
	TargetDate       string                       `json:"target_date"`
	Generated        bool                         `json:"generated"`
	RecommendedAt    time.Time                    `json:"recommended_at"`
	Recommendations  []RecommendationItemResponse `json:"recommendations"`
}
type DailyDiscoveryListResponse struct {
	Items      []DailyDiscoveryResponse `json:"items"`
	Page       int                      `json:"page"`
	Limit      int                      `json:"limit"`
	Total      int64                    `json:"total"`
	TotalPages int                      `json:"total_pages"`
}

func dailyResponse(v *entity.Recommendation, g bool) DailyDiscoveryResponse {
	base := toRecommendationResponse(v)
	d := ""
	if v.TargetDate != nil {
		d = v.TargetDate.Format("2006-01-02")
	}
	return DailyDiscoveryResponse{base.RecommendationID, d, g, base.RecommendedAt, base.Recommendations}
}
func (c *dailyDiscoveryController) GetToday(x echo.Context) error {
	id, ok := authUserID(x)
	if !ok {
		return writeAPIError(x, 401, "UNAUTHORIZED", "認証が必要です")
	}
	v, g, e := c.usecase.GetToday(x.Request().Context(), id)
	if e != nil {
		return c.handle(x, e)
	}
	return x.JSON(200, dailyResponse(v, g))
}
func (c *dailyDiscoveryController) List(x echo.Context) error {
	id, ok := authUserID(x)
	if !ok {
		return writeAPIError(x, 401, "UNAUTHORIZED", "認証が必要です")
	}
	page := 1
	limit := 10
	if value := x.QueryParam("page"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			return writeAPIError(x, 400, "INVALID_PAGE", "Page指定が不正です")
		}
		page = parsed
	}
	if value := x.QueryParam("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			return writeAPIError(x, 400, "INVALID_LIMIT", "Limit指定が不正です")
		}
		limit = parsed
	}
	v, e := c.usecase.List(x.Request().Context(), id, page, limit)
	if e != nil {
		return c.handle(x, e)
	}
	items := make([]DailyDiscoveryResponse, 0, len(v.Items))
	for i := range v.Items {
		items = append(items, dailyResponse(&v.Items[i], false))
	}
	return x.JSON(200, DailyDiscoveryListResponse{items, v.Page, v.Limit, v.Total, v.TotalPages})
}
func (c *dailyDiscoveryController) GetByID(x echo.Context) error {
	id, ok := authUserID(x)
	if !ok {
		return writeAPIError(x, 401, "UNAUTHORIZED", "認証が必要です")
	}
	rid, e := uuid.Parse(x.Param("recommendationId"))
	if e != nil {
		return writeAPIError(x, 400, "INVALID_RECOMMENDATION_ID", "Recommendation IDが不正です")
	}
	v, e := c.usecase.GetByID(x.Request().Context(), id, rid)
	if e != nil {
		return c.handle(x, e)
	}
	return x.JSON(200, dailyResponse(v, false))
}
func (c *dailyDiscoveryController) handle(x echo.Context, e error) error {
	switch {
	case errors.Is(e, entity.ErrQuestionAnswerRequired):
		return writeAPIError(x, 400, "QUESTION_ANSWER_REQUIRED", "質問回答を完了してください")
	case errors.Is(e, entity.ErrUserProfileRequired):
		return writeAPIError(x, 400, "USER_PROFILE_REQUIRED", "User Profileが必要です")
	case errors.Is(e, entity.ErrUserSuspended):
		return writeAPIError(x, 403, "USER_SUSPENDED", "利用停止中です")
	case errors.Is(e, entity.ErrDailyDiscoveryAccessDenied):
		return writeAPIError(x, 403, "DAILY_DISCOVERY_ACCESS_DENIED", "アクセスできません")
	case errors.Is(e, entity.ErrDailyDiscoveryNotFound):
		return writeAPIError(x, 404, "DAILY_DISCOVERY_NOT_FOUND", "Daily Discoveryが見つかりません")
	case errors.Is(e, entity.ErrDailyDiscoveryCandidatesNotFound):
		return writeAPIError(x, 422, "DAILY_DISCOVERY_CANDIDATES_NOT_FOUND", "推薦候補が不足しています")
	case errors.Is(e, entity.ErrUserNotFound):
		return writeAPIError(x, 404, "USER_NOT_FOUND", "Userが見つかりません")
	case errors.Is(e, entity.ErrDailyDiscoveryGenerationConflict):
		return writeAPIError(x, 409, "DAILY_DISCOVERY_GENERATION_CONFLICT", "同時生成の競合を解決できませんでした")
	case errors.Is(e, entity.ErrDuplicateRecommendationSong):
		return writeAPIError(x, http.StatusConflict, "DUPLICATE_RECOMMENDATION_SONG", "直近30日以内に推薦済みの楽曲が含まれています")
	case errors.Is(e, entity.ErrScoreCalculationFailed):
		return writeAPIError(x, http.StatusInternalServerError, "SCORE_CALCULATION_FAILED", "推薦スコアを計算できませんでした")
	case errors.Is(e, entity.ErrDailyDiscoveryGenerateFailed):
		return writeAPIError(x, 500, "DAILY_DISCOVERY_GENERATE_FAILED", "Daily Discoveryを生成できませんでした")
	case errors.Is(e, entity.ErrDailyDiscoverySaveFailed):
		return writeAPIError(x, 500, "DAILY_DISCOVERY_SAVE_FAILED", "保存できませんでした")
	default:
		return writeAPIError(x, http.StatusInternalServerError, "DAILY_DISCOVERY_FETCH_FAILED", "Daily Discoveryを取得できませんでした")
	}
}

package controller

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"
)

type RecommendationItemResponse struct {
	RecommendationDetailID uuid.UUID                 `json:"recommendation_detail_id"`
	RecommendationType     entity.RecommendationType `json:"recommendation_type"`
	SongID                 uuid.UUID                 `json:"song_id"`
	SongName               string                    `json:"song_name"`
	ArtistID               uuid.UUID                 `json:"artist_id"`
	ArtistName             string                    `json:"artist_name"`
	ArtworkURL             string                    `json:"artwork_url"`
	SpotifyURL             string                    `json:"spotify_url"`
	Reason                 string                    `json:"reason"`
	FeedbackStatus         *entity.FeedbackStatus    `json:"feedback_status"`
}
type RecommendationResponse struct {
	RecommendationID uuid.UUID                    `json:"recommendation_id"`
	RecommendedAt    time.Time                    `json:"recommended_at"`
	Recommendations  []RecommendationItemResponse `json:"recommendations"`
}
type IRecommendationController interface {
	Generate(echo.Context) error
	GetLatest(echo.Context) error
	GetByID(echo.Context) error
}
type recommendationController struct {
	usecase usecase.IRecommendationUsecase
}

func NewRecommendationController(value usecase.IRecommendationUsecase) IRecommendationController {
	return &recommendationController{usecase: value}
}
func (c *recommendationController) Generate(ctx echo.Context) error {
	id, ok := authUserID(ctx)
	if !ok {
		return writeAPIError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です")
	}
	value, err := c.usecase.Generate(ctx.Request().Context(), id)
	if err != nil {
		return c.handleError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, toRecommendationResponse(value))
}
func (c *recommendationController) GetLatest(ctx echo.Context) error {
	id, ok := authUserID(ctx)
	if !ok {
		return writeAPIError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です")
	}
	value, err := c.usecase.GetLatest(ctx.Request().Context(), id)
	if err != nil {
		return c.handleError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, toRecommendationResponse(value))
}
func (c *recommendationController) GetByID(ctx echo.Context) error {
	id, ok := authUserID(ctx)
	if !ok {
		return writeAPIError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です")
	}
	recommendationID, err := uuid.Parse(ctx.Param("recommendationId"))
	if err != nil {
		return c.handleError(ctx, entity.ErrInvalidRecommendationID)
	}
	value, err := c.usecase.GetByID(ctx.Request().Context(), id, recommendationID)
	if err != nil {
		return c.handleError(ctx, err)
	}
	return ctx.JSON(http.StatusOK, toRecommendationResponse(value))
}
func toRecommendationResponse(value *entity.Recommendation) RecommendationResponse {
	result := RecommendationResponse{RecommendationID: value.ID, RecommendedAt: value.RecommendedAt, Recommendations: make([]RecommendationItemResponse, 0, len(value.Details))}
	for _, detail := range value.Details {
		result.Recommendations = append(result.Recommendations, RecommendationItemResponse{RecommendationDetailID: detail.ID, RecommendationType: detail.Type, SongID: detail.SongID, SongName: detail.Song.Name, ArtistID: detail.Song.ArtistID, ArtistName: detail.Song.Artist.Name, ArtworkURL: detail.Song.ImageURL, SpotifyURL: detail.Song.SpotifyURL, Reason: detail.Reason, FeedbackStatus: detail.FeedbackStatus})
	}
	return result
}
func (c *recommendationController) handleError(ctx echo.Context, err error) error {
	switch {
	case errors.Is(err, entity.ErrQuestionAnswerRequired):
		return writeAPIError(ctx, 400, "QUESTION_ANSWER_REQUIRED", "質問への回答を完了してください")
	case errors.Is(err, entity.ErrUserProfileRequired):
		return writeAPIError(ctx, 400, "USER_PROFILE_REQUIRED", "ユーザープロフィールが必要です")
	case errors.Is(err, entity.ErrInvalidRecommendationID):
		return writeAPIError(ctx, 400, "INVALID_RECOMMENDATION_ID", "推薦IDが不正です")
	case errors.Is(err, entity.ErrUserSuspended):
		return writeAPIError(ctx, 403, "USER_SUSPENDED", "このアカウントは現在利用できません")
	case errors.Is(err, entity.ErrRecommendationAccessDenied):
		return writeAPIError(ctx, 403, "RECOMMENDATION_ACCESS_DENIED", "この推薦結果へアクセスできません")
	case errors.Is(err, entity.ErrRecommendationNotFound):
		return writeAPIError(ctx, 404, "RECOMMENDATION_NOT_FOUND", "推薦結果が見つかりません")
	case errors.Is(err, entity.ErrRecommendationAlreadyGenerated):
		return writeAPIError(ctx, 409, "RECOMMENDATION_ALREADY_GENERATED", "推薦結果はすでに生成されています")
	case errors.Is(err, entity.ErrRecommendationCandidatesImportedPendingReview):
		return writeAPIError(ctx, 422, "RECOMMENDATION_CANDIDATES_IMPORTED_PENDING_REVIEW", "Spotifyから候補を取り込みました。管理者によるTag確認後に推薦できます")
	case errors.Is(err, entity.ErrSpotifyCandidateImportFailed):
		return writeAPIError(ctx, 503, "SPOTIFY_CANDIDATE_IMPORT_FAILED", "Spotifyから推薦候補を取得できませんでした")
	case errors.Is(err, entity.ErrRecommendationCandidatesNotFound):
		return writeAPIError(ctx, 422, "RECOMMENDATION_CANDIDATES_NOT_FOUND", "推薦候補が不足しています")
	case errors.Is(err, entity.ErrInvalidWeightConfig):
		return writeAPIError(ctx, http.StatusInternalServerError, "INVALID_WEIGHT_CONFIG", "推薦設定が不正です")
	case errors.Is(err, entity.ErrInvalidGenreRelation):
		return writeAPIError(ctx, http.StatusInternalServerError, "INVALID_GENRE_RELATION", "ジャンル関連設定が不正です")
	case errors.Is(err, entity.ErrInvalidRecommendationType):
		return writeAPIError(ctx, http.StatusInternalServerError, "INVALID_RECOMMENDATION_TYPE", "推薦種別が不正です")
	case errors.Is(err, entity.ErrRecommendationSelectionFailed):
		return writeAPIError(ctx, http.StatusUnprocessableEntity, "RECOMMENDATION_SELECTION_FAILED", "3曲を選定できませんでした")
	case errors.Is(err, entity.ErrDuplicateRecommendationSong):
		return writeAPIError(ctx, http.StatusConflict, "DUPLICATE_RECOMMENDATION_SONG", "直近30日以内に推薦済みの楽曲が含まれています")
	case errors.Is(err, entity.ErrScoreCalculationFailed):
		return writeAPIError(ctx, http.StatusInternalServerError, "SCORE_CALCULATION_FAILED", "推薦スコアを計算できませんでした")
	case errors.Is(err, entity.ErrRecommendationReasonGenerateFailed):
		return writeAPIError(ctx, http.StatusInternalServerError, "RECOMMENDATION_REASON_GENERATE_FAILED", "推薦理由を生成できませんでした")
	case errors.Is(err, entity.ErrRecommendationSaveFailed):
		return writeAPIError(ctx, 500, "RECOMMENDATION_SAVE_FAILED", "推薦結果を保存できませんでした")
	case errors.Is(err, entity.ErrRecommendationGenerateFailed):
		return writeAPIError(ctx, 500, "RECOMMENDATION_GENERATE_FAILED", "推薦結果を生成できませんでした")
	default:
		return writeAPIError(ctx, 500, "INTERNAL_SERVER_ERROR", "サーバー内部でエラーが発生しました")
	}
}

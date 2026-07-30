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

type SaveFeedbackRequest struct {
	FeedbackType entity.FeedbackType `json:"feedback_type"`
}
type SaveFeedbackResponse struct {
	FeedbackID             uuid.UUID           `json:"feedback_id"`
	RecommendationDetailID uuid.UUID           `json:"recommendation_detail_id"`
	FeedbackType           entity.FeedbackType `json:"feedback_type"`
	Created                bool                `json:"created"`
	UpdatedAt              time.Time           `json:"updated_at"`
}
type FeedbackDetailResponse struct {
	RecommendationDetailID uuid.UUID  `json:"recommendation_detail_id"`
	FeedbackStatus         string     `json:"feedback_status"`
	UpdatedAt              *time.Time `json:"updated_at"`
}
type IFeedbackController interface {
	Save(echo.Context) error
	Get(echo.Context) error
}
type feedbackController struct{ usecase usecase.IFeedbackUsecase }

func NewFeedbackController(value usecase.IFeedbackUsecase) IFeedbackController {
	return &feedbackController{usecase: value}
}
func (c *feedbackController) parse(ctx echo.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(ctx.Param("recommendationDetailId"))
	if err != nil {
		return uuid.Nil, entity.ErrInvalidRecommendationDetailID
	}
	return id, nil
}
func (c *feedbackController) Save(ctx echo.Context) error {
	uid, ok := authUserID(ctx)
	if !ok {
		return writeAPIError(ctx, 401, "UNAUTHORIZED", "認証が必要です")
	}
	detailID, err := c.parse(ctx)
	if err != nil {
		return c.handle(ctx, err)
	}
	var request SaveFeedbackRequest
	if err := ctx.Bind(&request); err != nil {
		return writeAPIError(ctx, 400, "INVALID_REQUEST", "Request形式が不正です")
	}
	result, err := c.usecase.Save(ctx.Request().Context(), uid, usecase.SaveFeedbackInput{RecommendationDetailID: detailID, FeedbackType: request.FeedbackType})
	if err != nil {
		return c.handle(ctx, err)
	}
	return ctx.JSON(http.StatusOK, SaveFeedbackResponse{FeedbackID: result.Feedback.ID, RecommendationDetailID: result.Feedback.RecommendationDetailID, FeedbackType: result.Feedback.Type, Created: result.Created, UpdatedAt: result.Feedback.UpdatedAt})
}
func (c *feedbackController) Get(ctx echo.Context) error {
	uid, ok := authUserID(ctx)
	if !ok {
		return writeAPIError(ctx, 401, "UNAUTHORIZED", "認証が必要です")
	}
	detailID, err := c.parse(ctx)
	if err != nil {
		return c.handle(ctx, err)
	}
	value, err := c.usecase.Get(ctx.Request().Context(), uid, detailID)
	if err != nil {
		return c.handle(ctx, err)
	}
	return ctx.JSON(http.StatusOK, FeedbackDetailResponse{RecommendationDetailID: value.RecommendationDetailID, FeedbackStatus: value.Status, UpdatedAt: value.UpdatedAt})
}
func (c *feedbackController) handle(ctx echo.Context, err error) error {
	switch {
	case errors.Is(err, entity.ErrInvalidRecommendationDetailID):
		return writeAPIError(ctx, 400, "INVALID_RECOMMENDATION_DETAIL_ID", "推薦詳細IDが不正です")
	case errors.Is(err, entity.ErrInvalidFeedbackType):
		return writeAPIError(ctx, 400, "INVALID_FEEDBACK_TYPE", "Feedback Typeが不正です")
	case errors.Is(err, entity.ErrUserSuspended):
		return writeAPIError(ctx, 403, "USER_SUSPENDED", "このアカウントは現在利用できません")
	case errors.Is(err, entity.ErrFeedbackAccessDenied):
		return writeAPIError(ctx, 403, "FEEDBACK_ACCESS_DENIED", "この推薦結果へFeedbackを登録できません")
	case errors.Is(err, entity.ErrUserNotFound):
		return writeAPIError(ctx, 404, "USER_NOT_FOUND", "Userが存在しません")
	case errors.Is(err, entity.ErrRecommendationDetailNotFound):
		return writeAPIError(ctx, 404, "RECOMMENDATION_DETAIL_NOT_FOUND", "推薦詳細が見つかりません")
	case errors.Is(err, entity.ErrFeedbackSaveFailed):
		return writeAPIError(ctx, 500, "FEEDBACK_SAVE_FAILED", "Feedbackを保存できませんでした")
	case errors.Is(err, entity.ErrFeedbackUpdateFailed):
		return writeAPIError(ctx, 500, "FEEDBACK_UPDATE_FAILED", "Feedbackを変更できませんでした")
	default:
		return writeAPIError(ctx, 500, "INTERNAL_SERVER_ERROR", "サーバー内部でエラーが発生しました")
	}
}

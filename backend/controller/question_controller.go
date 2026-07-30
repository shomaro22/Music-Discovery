package controller

import (
	"errors"
	"net/http"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type QuestionChoiceResponse struct {
	ID        uuid.UUID `json:"id"`
	Code      string    `json:"code"`
	Label     string    `json:"label"`
	SortOrder int       `json:"sort_order"`
}
type QuestionResponse struct {
	ID            uuid.UUID                 `json:"id"`
	Code          string                    `json:"code"`
	Title         string                    `json:"title"`
	AnswerType    entity.QuestionAnswerType `json:"answer_type"`
	MinSelections int                       `json:"min_selections"`
	MaxSelections *int                      `json:"max_selections"`
	SortOrder     int                       `json:"sort_order"`
	Choices       []QuestionChoiceResponse  `json:"choices"`
}
type QuestionListResponse struct {
	Questions []QuestionResponse `json:"questions"`
}
type SaveQuestionAnswerRequest struct {
	QuestionID uuid.UUID   `json:"question_id"`
	ChoiceIDs  []uuid.UUID `json:"choice_ids"`
}
type SaveQuestionAnswersRequest struct {
	Answers []SaveQuestionAnswerRequest `json:"answers"`
}
type SaveQuestionAnswersResponse struct {
	AnswerSessionID    uuid.UUID `json:"answer_session_id"`
	ProfileInitialized bool      `json:"profile_initialized"`
	AnsweredAt         time.Time `json:"answered_at"`
}
type LatestQuestionAnswerResponse struct {
	QuestionID uuid.UUID   `json:"question_id"`
	ChoiceIDs  []uuid.UUID `json:"choice_ids"`
}
type LatestQuestionAnswersResponse struct {
	Answers    []LatestQuestionAnswerResponse `json:"answers"`
	AnsweredAt *time.Time                     `json:"answered_at,omitempty"`
}

type IQuestionController interface {
	ListQuestions(echo.Context) error
	SaveAnswers(echo.Context) error
	GetLatestAnswers(echo.Context) error
}
type questionController struct{ questionUsecase usecase.IQuestionUsecase }

func NewQuestionController(questionUsecase usecase.IQuestionUsecase) IQuestionController {
	return &questionController{questionUsecase: questionUsecase}
}

func (c *questionController) ListQuestions(ctx echo.Context) error {
	questions, err := c.questionUsecase.ListQuestions(ctx.Request().Context())
	if err != nil {
		return c.handleError(ctx, err)
	}
	response := QuestionListResponse{Questions: make([]QuestionResponse, 0, len(questions))}
	for _, question := range questions {
		item := QuestionResponse{ID: question.ID, Code: question.Code, Title: question.Title, AnswerType: question.AnswerType, MinSelections: question.MinSelections, MaxSelections: question.MaxSelections, SortOrder: question.SortOrder, Choices: make([]QuestionChoiceResponse, 0, len(question.Choices))}
		for _, choice := range question.Choices {
			item.Choices = append(item.Choices, QuestionChoiceResponse{ID: choice.ID, Code: choice.Code, Label: choice.Label, SortOrder: choice.SortOrder})
		}
		response.Questions = append(response.Questions, item)
	}
	return ctx.JSON(http.StatusOK, response)
}

func (c *questionController) SaveAnswers(ctx echo.Context) error {
	userID, ok := authUserID(ctx)
	if !ok {
		return writeAPIError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です")
	}
	var request SaveQuestionAnswersRequest
	if err := ctx.Bind(&request); err != nil {
		return writeAPIError(ctx, http.StatusBadRequest, "INVALID_REQUEST", "入力内容を確認してください")
	}
	input := usecase.SaveQuestionAnswersInput{Answers: make([]usecase.QuestionAnswerInput, 0, len(request.Answers))}
	for _, answer := range request.Answers {
		input.Answers = append(input.Answers, usecase.QuestionAnswerInput{QuestionID: answer.QuestionID, ChoiceIDs: answer.ChoiceIDs})
	}
	result, err := c.questionUsecase.SaveAnswers(ctx.Request().Context(), userID, input)
	if err != nil {
		return c.handleError(ctx, err)
	}
	return ctx.JSON(http.StatusCreated, SaveQuestionAnswersResponse{AnswerSessionID: result.AnswerSessionID, ProfileInitialized: result.ProfileInitialized, AnsweredAt: result.AnsweredAt})
}

func (c *questionController) GetLatestAnswers(ctx echo.Context) error {
	userID, ok := authUserID(ctx)
	if !ok {
		return writeAPIError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です")
	}
	result, err := c.questionUsecase.GetLatestAnswers(ctx.Request().Context(), userID)
	if err != nil {
		return c.handleError(ctx, err)
	}
	response := LatestQuestionAnswersResponse{Answers: make([]LatestQuestionAnswerResponse, 0, len(result.Answers)), AnsweredAt: result.AnsweredAt}
	for _, answer := range result.Answers {
		response.Answers = append(response.Answers, LatestQuestionAnswerResponse{QuestionID: answer.QuestionID, ChoiceIDs: answer.ChoiceIDs})
	}
	return ctx.JSON(http.StatusOK, response)
}

func (c *questionController) handleError(ctx echo.Context, err error) error {
	switch {
	case errors.Is(err, entity.ErrRequiredQuestionMissing):
		return writeAPIError(ctx, http.StatusBadRequest, "REQUIRED_QUESTION_MISSING", "必須の質問に回答してください")
	case errors.Is(err, entity.ErrDuplicateQuestion):
		return writeAPIError(ctx, http.StatusBadRequest, "DUPLICATE_QUESTION", "質問が重複しています")
	case errors.Is(err, entity.ErrInvalidSelectionCount):
		return writeAPIError(ctx, http.StatusBadRequest, "INVALID_SELECTION_COUNT", "選択数が条件を満たしていません")
	case errors.Is(err, entity.ErrDuplicateChoice):
		return writeAPIError(ctx, http.StatusBadRequest, "DUPLICATE_CHOICE", "選択肢が重複しています")
	case errors.Is(err, entity.ErrQuestionNotFound):
		return writeAPIError(ctx, http.StatusNotFound, "QUESTION_NOT_FOUND", "質問が見つかりません")
	case errors.Is(err, entity.ErrChoiceNotFound):
		return writeAPIError(ctx, http.StatusNotFound, "CHOICE_NOT_FOUND", "選択肢が見つかりません")
	case errors.Is(err, entity.ErrQuestionChoiceMismatch):
		return writeAPIError(ctx, http.StatusConflict, "QUESTION_CHOICE_MISMATCH", "質問と選択肢が一致しません")
	case errors.Is(err, entity.ErrUserSuspended):
		return writeAPIError(ctx, http.StatusForbidden, "USER_SUSPENDED", "このアカウントは現在利用できません")
	case errors.Is(err, entity.ErrAnswerSaveFailed):
		return writeAPIError(ctx, http.StatusInternalServerError, "ANSWER_SAVE_FAILED", "回答を保存できませんでした")
	case errors.Is(err, entity.ErrProfileCreateFailed):
		return writeAPIError(ctx, http.StatusInternalServerError, "PROFILE_CREATE_FAILED", "プロフィールを更新できませんでした")
	default:
		return writeAPIError(ctx, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "サーバー内部でエラーが発生しました")
	}
}

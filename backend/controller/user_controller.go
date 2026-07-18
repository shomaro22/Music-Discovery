package controller

import (
	"errors"
	"net/http"

	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"
	"music-discovery/backend/validator"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type SignUpRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignUpResponse struct {
	ID        uuid.UUID         `json:"id"`
	Name      string            `json:"name"`
	Email     string            `json:"email"`
	Role      entity.UserRole   `json:"role"`
	Status    entity.UserStatus `json:"status"`
	CreatedAt string            `json:"created_at"`
}

type IUserController interface {
	SignUp(c echo.Context) error
}

type userController struct {
	userUsecase usecase.IUserUsecase
}

func NewUserController(userUsecase usecase.IUserUsecase) IUserController {
	return &userController{userUsecase: userUsecase}
}

func (c *userController) SignUp(ctx echo.Context) error {
	var request SignUpRequest
	if err := ctx.Bind(&request); err != nil {
		return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "入力内容を確認してください")
	}

	values, validationErrors := validator.ValidateSignUp(request.Name, request.Email, request.Password)
	if len(validationErrors) > 0 {
		return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "入力内容を確認してください")
	}

	user, err := c.userUsecase.SignUp(ctx.Request().Context(), usecase.SignUpInput{
		Name: values.Name, Email: values.Email, Password: values.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrEmailAlreadyExists):
			return writeAPIError(ctx, http.StatusConflict, "EMAIL_ALREADY_EXISTS", "このメールアドレスは既に登録されています")
		case errors.Is(err, entity.ErrInvalidInput):
			return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "入力内容を確認してください")
		default:
			return writeAPIError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "サーバー内部でエラーが発生しました")
		}
	}

	return ctx.JSON(http.StatusCreated, SignUpResponse{
		ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role, Status: user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func writeAPIError(ctx echo.Context, status int, code, message string) error {
	return ctx.JSON(status, entity.APIErrorResponse{
		Status: status, Code: code, Message: message,
		RequestID: ctx.Response().Header().Get(echo.HeaderXRequestID),
	})
}

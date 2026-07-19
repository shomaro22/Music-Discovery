package controller

import (
	"errors"
	"net/http"
	"time"

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
type LoginRequest struct {
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
type LoginUserResponse struct {
	ID     uuid.UUID         `json:"id"`
	Name   string            `json:"name"`
	Email  string            `json:"email"`
	Role   entity.UserRole   `json:"role"`
	Status entity.UserStatus `json:"status"`
}
type LoginResponse struct {
	AccessToken string            `json:"access_token"`
	TokenType   string            `json:"token_type"`
	ExpiresIn   int64             `json:"expires_in"`
	User        LoginUserResponse `json:"user"`
}

type IUserController interface {
	SignUp(c echo.Context) error
	Login(c echo.Context) error
}
type userController struct {
	userUsecase  usecase.IUserUsecase
	cookieConfig CookieConfig
}

func NewUserController(userUsecase usecase.IUserUsecase, optional ...CookieConfig) IUserController {
	cfg, _ := LoadCookieConfigFromEnv()
	if len(optional) > 0 {
		cfg = optional[0]
	}
	return &userController{userUsecase: userUsecase, cookieConfig: cfg}
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
	user, err := c.userUsecase.SignUp(ctx.Request().Context(), usecase.SignUpInput{Name: values.Name, Email: values.Email, Password: values.Password})
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
	return ctx.JSON(http.StatusCreated, SignUpResponse{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role, Status: user.Status, CreatedAt: user.CreatedAt.Format(time.RFC3339)})
}

func (c *userController) Login(ctx echo.Context) error {
	var request LoginRequest
	if err := ctx.Bind(&request); err != nil {
		return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "入力内容を確認してください")
	}
	values, validationErrors := validator.ValidateLogin(request.Email, request.Password)
	if len(validationErrors) > 0 {
		return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "入力内容を確認してください")
	}
	result, err := c.userUsecase.Login(ctx.Request().Context(), usecase.LoginInput{Email: values.Email, Password: values.Password})
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrInvalidCredentials):
			return writeAPIError(ctx, http.StatusUnauthorized, "INVALID_CREDENTIALS", "メールアドレスまたはパスワードが正しくありません。")
		case errors.Is(err, entity.ErrUserSuspended):
			return writeAPIError(ctx, http.StatusForbidden, "USER_SUSPENDED", "このアカウントは現在利用できません。")
		case errors.Is(err, entity.ErrInvalidInput):
			return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "入力内容を確認してください")
		default:
			return echo.NewHTTPError(http.StatusInternalServerError, "サーバー内部でエラーが発生しました").SetInternal(err)
		}
	}
	setAuthCookies(ctx, result, c.cookieConfig)
	return ctx.JSON(http.StatusOK, LoginResponse{AccessToken: result.AccessToken, TokenType: "Bearer", ExpiresIn: result.ExpiresIn, User: LoginUserResponse{ID: result.User.ID, Name: result.User.Name, Email: result.User.Email, Role: result.User.Role, Status: result.User.Status}})
}

func setAuthCookies(ctx echo.Context, result *usecase.LoginResult, cfg CookieConfig) {
	maxAge := int(time.Until(result.RefreshTokenExpiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	ctx.SetCookie(&http.Cookie{Name: cfg.RefreshName, Value: result.RefreshToken, Path: cfg.Path, Domain: cfg.Domain, HttpOnly: true, Secure: cfg.Secure, SameSite: cfg.SameSite, Expires: result.RefreshTokenExpiresAt, MaxAge: maxAge})
	ctx.SetCookie(&http.Cookie{Name: cfg.CSRFName, Value: result.CSRFToken, Path: cfg.Path, Domain: cfg.Domain, HttpOnly: false, Secure: cfg.Secure, SameSite: cfg.SameSite, Expires: result.RefreshTokenExpiresAt, MaxAge: maxAge})
}

func writeAPIError(ctx echo.Context, status int, code, message string) error {
	return ctx.JSON(status, entity.APIErrorResponse{Status: status, Code: code, Message: message, RequestID: ctx.Response().Header().Get(echo.HeaderXRequestID)})
}

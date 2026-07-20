package controller

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/usecase"

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

type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

type IUserController interface {
	SignUp(c echo.Context) error
	Login(c echo.Context) error
	Refresh(c echo.Context) error
	CurrentUser(c echo.Context) error
}
type userController struct{ userUsecase usecase.IUserUsecase }

func NewUserController(userUsecase usecase.IUserUsecase) IUserController {
	return &userController{userUsecase: userUsecase}
}

func (c *userController) SignUp(ctx echo.Context) error {
	var request SignUpRequest
	if err := ctx.Bind(&request); err != nil {
		return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "入力内容を確認してください")
	}
	user, err := c.userUsecase.SignUp(ctx.Request().Context(), usecase.SignUpInput{Name: request.Name, Email: request.Email, Password: request.Password})
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
	result, err := c.userUsecase.Login(ctx.Request().Context(), usecase.LoginInput{Email: request.Email, Password: request.Password})
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrInvalidCredentials):
			return writeAPIError(ctx, http.StatusUnauthorized, "INVALID_CREDENTIALS", "メールアドレスまたはパスワードが正しくありません。")
		case errors.Is(err, entity.ErrUserSuspended):
			return writeAPIError(ctx, http.StatusForbidden, "USER_SUSPENDED", "このアカウントは現在利用できません。")
		case errors.Is(err, entity.ErrInvalidInput):
			return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "入力内容を確認してください")
		default:
			return writeAPIError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "サーバー内部でエラーが発生しました")
		}
	}
	setAuthCookies(ctx, result.RefreshToken, result.CSRFToken, result.RefreshTokenExpiresAt)
	return ctx.JSON(http.StatusOK, LoginResponse{AccessToken: result.AccessToken, TokenType: "Bearer", ExpiresIn: result.ExpiresIn, User: LoginUserResponse{ID: result.User.ID, Name: result.User.Name, Email: result.User.Email, Role: result.User.Role, Status: result.User.Status}})
}

func (c *userController) Refresh(ctx echo.Context) error {
	refreshName := cookieName("REFRESH_TOKEN_COOKIE_NAME", "refresh_token")
	cookie, err := ctx.Cookie(refreshName)
	if err != nil || cookie.Value == "" {
		return writeAPIError(ctx, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Refresh Tokenが無効です")
	}
	result, err := c.userUsecase.Refresh(ctx.Request().Context(), cookie.Value)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrRefreshTokenReused):
			clearAuthCookies(ctx)
			return writeAPIError(ctx, http.StatusUnauthorized, "REFRESH_TOKEN_REUSED", "Refresh Tokenの再利用を検知しました")
		case errors.Is(err, entity.ErrInvalidRefreshToken):
			clearAuthCookies(ctx)
			return writeAPIError(ctx, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Refresh Tokenが無効です")
		case errors.Is(err, entity.ErrUserSuspended):
			clearAuthCookies(ctx)
			return writeAPIError(ctx, http.StatusForbidden, "USER_SUSPENDED", "このアカウントは現在利用できません。")
		default:
			return writeAPIError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "サーバー内部でエラーが発生しました")
		}
	}
	setAuthCookies(ctx, result.RefreshToken, result.CSRFToken, result.RefreshTokenExpiresAt)
	return ctx.JSON(http.StatusOK, RefreshResponse{AccessToken: result.AccessToken, TokenType: "Bearer", ExpiresIn: result.ExpiresIn})
}

func setAuthCookies(ctx echo.Context, refreshToken, csrfToken string, expiresAt time.Time) {
	secure := strings.EqualFold(os.Getenv("APP_ENV"), "production")
	sameSite := http.SameSiteLaxMode
	if strings.EqualFold(os.Getenv("COOKIE_SAME_SITE"), "strict") {
		sameSite = http.SameSiteStrictMode
	}
	refreshName := cookieName("REFRESH_TOKEN_COOKIE_NAME", "refresh_token")
	csrfName := cookieName("CSRF_COOKIE_NAME", "csrf_token")
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	ctx.SetCookie(&http.Cookie{Name: refreshName, Value: refreshToken, Path: "/api/auth", HttpOnly: true, Secure: secure, SameSite: sameSite, Expires: expiresAt, MaxAge: maxAge})
	ctx.SetCookie(&http.Cookie{Name: csrfName, Value: csrfToken, Path: "/", HttpOnly: false, Secure: secure, SameSite: sameSite, Expires: expiresAt, MaxAge: maxAge})
}

func clearAuthCookies(ctx echo.Context) {
	secure := strings.EqualFold(os.Getenv("APP_ENV"), "production")
	sameSite := http.SameSiteLaxMode
	if strings.EqualFold(os.Getenv("COOKIE_SAME_SITE"), "strict") {
		sameSite = http.SameSiteStrictMode
	}
	expired := time.Unix(0, 0)
	ctx.SetCookie(&http.Cookie{
		Name:     cookieName("REFRESH_TOKEN_COOKIE_NAME", "refresh_token"),
		Value:    "",
		Path:     "/api/auth",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Expires:  expired,
		MaxAge:   -1,
	})
	ctx.SetCookie(&http.Cookie{
		Name:     cookieName("CSRF_COOKIE_NAME", "csrf_token"),
		Value:    "",
		Path:     "/",
		HttpOnly: false,
		Secure:   secure,
		SameSite: sameSite,
		Expires:  expired,
		MaxAge:   -1,
	})
}

func cookieName(envName, fallback string) string {
	if value := os.Getenv(envName); value != "" {
		return value
	}
	return fallback
}

func writeAPIError(ctx echo.Context, status int, code, message string) error {
	return ctx.JSON(status, entity.APIErrorResponse{Status: status, Code: code, Message: message, RequestID: ctx.Response().Header().Get(echo.HeaderXRequestID)})
}

func (c *userController) CurrentUser(ctx echo.Context) error {
	user, ok := ctx.Get("auth_user").(*entity.User)
	if !ok || user == nil {
		return writeAPIError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です")
	}
	return ctx.JSON(http.StatusOK, LoginUserResponse{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role, Status: user.Status})
}

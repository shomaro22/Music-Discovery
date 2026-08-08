package controller

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/middleware"
	"music-discovery/backend/usecase"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type SignUpRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
type UpdateProfileRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}
type WithdrawRequest struct {
	CurrentPassword string `json:"current_password"`
}
type WithdrawResponse struct {
	Message string `json:"message"`
}
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type SignUpUserResponse struct {
	ID        uuid.UUID         `json:"id"`
	Name      string            `json:"name"`
	Email     string            `json:"email"`
	Role      entity.UserRole   `json:"role"`
	Status    entity.UserStatus `json:"status"`
	CreatedAt time.Time         `json:"created_at"`
}
type SignUpResponse struct {
	AccessToken string             `json:"access_token"`
	TokenType   string             `json:"token_type"`
	ExpiresIn   int64              `json:"expires_in"`
	User        SignUpUserResponse `json:"user"`
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

type apiErrorResponse struct {
	Status    int    `json:"status"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

type IUserController interface {
	SignUp(c echo.Context) error
	Login(c echo.Context) error
	Refresh(c echo.Context) error
	Logout(c echo.Context) error
	CurrentUser(c echo.Context) error
	UpdateProfile(c echo.Context) error
	Withdraw(c echo.Context) error
	RequestPasswordReset(c echo.Context) error
	ConfirmPasswordReset(c echo.Context) error
}
type userController struct {
	userUsecase          usecase.IUserUsecase
	passwordResetUsecase usecase.IPasswordResetUsecase
}

type UserControllerOption func(*userController)

func WithPasswordResetUsecase(passwordResetUsecase usecase.IPasswordResetUsecase) UserControllerOption {
	return func(controller *userController) { controller.passwordResetUsecase = passwordResetUsecase }
}

func NewUserController(userUsecase usecase.IUserUsecase, options ...UserControllerOption) IUserController {
	controller := &userController{userUsecase: userUsecase}
	for _, option := range options {
		option(controller)
	}
	return controller
}

type PasswordResetRequest struct {
	Email string `json:"email"`
}

type PasswordResetConfirmRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

type PasswordResetRequestResponse struct {
	Message    string `json:"message"`
	ResetToken string `json:"reset_token,omitempty"`
}

type PasswordResetConfirmResponse struct {
	Message string `json:"message"`
}

func (c *userController) RequestPasswordReset(ctx echo.Context) error {
	if c.passwordResetUsecase == nil {
		return writeAPIError(ctx, http.StatusNotImplemented, "PASSWORD_RESET_UNAVAILABLE", "パスワード再設定を利用できません")
	}
	var request PasswordResetRequest
	if err := ctx.Bind(&request); err != nil {
		return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "入力内容を確認してください")
	}
	result, err := c.passwordResetUsecase.Request(ctx.Request().Context(), request.Email)
	if err != nil {
		if errors.Is(err, entity.ErrInvalidInput) {
			return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "有効なメールアドレスを入力してください")
		}
		return writeAPIError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "パスワード再設定を受け付けられませんでした")
	}
	response := PasswordResetRequestResponse{Message: "登録済みのメールアドレスの場合、パスワード再設定の案内を送信しました。"}
	if !strings.EqualFold(os.Getenv("APP_ENV"), "production") && result != nil {
		response.ResetToken = result.Token
	}
	return ctx.JSON(http.StatusAccepted, response)
}

func (c *userController) ConfirmPasswordReset(ctx echo.Context) error {
	if c.passwordResetUsecase == nil {
		return writeAPIError(ctx, http.StatusNotImplemented, "PASSWORD_RESET_UNAVAILABLE", "パスワード再設定を利用できません")
	}
	var request PasswordResetConfirmRequest
	if err := ctx.Bind(&request); err != nil {
		return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "入力内容を確認してください")
	}
	err := c.passwordResetUsecase.Confirm(ctx.Request().Context(), request.Token, request.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrInvalidPasswordResetToken):
			return writeAPIError(ctx, http.StatusBadRequest, "INVALID_PASSWORD_RESET_TOKEN", "再設定リンクが無効か、有効期限が切れています")
		case errors.Is(err, entity.ErrInvalidInput):
			return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "パスワードは8文字以上72バイト以内で入力してください")
		default:
			return writeAPIError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "パスワードを再設定できませんでした")
		}
	}
	clearAuthCookies(ctx)
	return ctx.JSON(http.StatusOK, PasswordResetConfirmResponse{Message: "パスワードを変更しました。新しいパスワードでログインしてください。"})
}

func (c *userController) SignUp(ctx echo.Context) error {
	var request SignUpRequest
	if err := ctx.Bind(&request); err != nil {
		return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "入力内容を確認してください")
	}
	result, err := c.userUsecase.SignUp(ctx.Request().Context(), usecase.SignUpInput{Name: request.Name, Email: request.Email, Password: request.Password})
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
	setAuthCookies(ctx, result.RefreshToken, result.CSRFToken, result.RefreshTokenExpiresAt)
	ctx.Set(middleware.SystemLogEventKey, "signup_succeeded")
	return ctx.JSON(http.StatusCreated, SignUpResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   result.ExpiresIn,
		User: SignUpUserResponse{
			ID:        result.User.ID,
			Name:      result.User.Name,
			Email:     result.User.Email,
			Role:      result.User.Role,
			Status:    result.User.Status,
			CreatedAt: result.User.CreatedAt,
		},
	})
}

func (c *userController) Login(ctx echo.Context) error {
	var request LoginRequest
	if err := ctx.Bind(&request); err != nil {
		return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "入力内容を確認してください")
	}
	result, err := c.userUsecase.Login(ctx.Request().Context(),
		usecase.LoginInput{Email: request.Email, Password: request.Password})
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrInvalidCredentials):
			ctx.Set(middleware.SystemLogEventKey, "login_failed")
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
	ctx.Set(middleware.SystemLogEventKey, "login_succeeded")
	return ctx.JSON(http.StatusOK, LoginResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer", ExpiresIn: result.ExpiresIn,
		User: LoginUserResponse{
			ID:     result.User.ID,
			Name:   result.User.Name,
			Email:  result.User.Email,
			Role:   result.User.Role,
			Status: result.User.Status}})
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
			ctx.Set(middleware.SystemLogEventKey, "refresh_token_reuse_detected")
			ctx.Set(middleware.SystemLogErrorCodeKey, "REFRESH_TOKEN_REUSED")
			ctx.Set(middleware.SystemLogMessageKey, "refresh token reuse detected")
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
	ctx.Set(middleware.SystemLogEventKey, "refresh_token_rotated")
	return ctx.JSON(http.StatusOK, RefreshResponse{AccessToken: result.AccessToken, TokenType: "Bearer", ExpiresIn: result.ExpiresIn})
}

func (c *userController) Logout(ctx echo.Context) error {
	refreshName := cookieName("REFRESH_TOKEN_COOKIE_NAME", "refresh_token")
	refreshToken := ""
	if cookie, err := ctx.Cookie(refreshName); err == nil {
		refreshToken = cookie.Value
	}
	if err := c.userUsecase.Logout(ctx.Request().Context(), refreshToken); err != nil {
		return writeAPIError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "サーバー内部でエラーが発生しました")
	}
	clearAuthCookies(ctx)
	return ctx.NoContent(http.StatusNoContent)
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
	ctx.Set(middleware.SystemLogErrorCodeKey, code)
	ctx.Set(middleware.SystemLogMessageKey, message)
	path := ctx.Request().URL.Path
	if ctx.Get(middleware.SystemLogEventKey) == nil {
		switch {
		case strings.Contains(path, "/recommendations") && ctx.Request().Method == http.MethodPost:
			ctx.Set(middleware.SystemLogEventKey, "recommendation_failed")
		case strings.Contains(path, "/daily-discovery"):
			ctx.Set(middleware.SystemLogEventKey, "daily_discovery_failed")
		case strings.Contains(path, "/spotify/"):
			ctx.Set(middleware.SystemLogEventKey, "spotify_api_failed")
			ctx.Set(middleware.SystemLogExternalServiceKey, "spotify")
		}
	}
	return ctx.JSON(status, apiErrorResponse{
		Status:    status,
		Code:      code,
		Message:   message,
		RequestID: ctx.Response().Header().Get(echo.HeaderXRequestID)})
}

func (c *userController) CurrentUser(ctx echo.Context) error {
	userID, ok := ctx.Get(middleware.AuthUserIDKey).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return writeAPIError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です")
	}

	user, err := c.userUsecase.CurrentUser(ctx.Request().Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrInvalidCredentials), errors.Is(err, entity.ErrUserNotFound):
			return writeAPIError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です")
		case errors.Is(err, entity.ErrUserSuspended):
			return writeAPIError(ctx, http.StatusForbidden, "USER_SUSPENDED", "このアカウントは現在利用できません。")
		default:
			return writeAPIError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "サーバー内部でエラーが発生しました")
		}
	}

	return ctx.JSON(http.StatusOK, LoginUserResponse{
		ID:     user.ID,
		Name:   user.Name,
		Email:  user.Email,
		Role:   user.Role,
		Status: user.Status})
}

func (c *userController) Withdraw(ctx echo.Context) error {
	userID, ok := ctx.Get(middleware.AuthUserIDKey).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return writeAPIError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です")
	}
	var request WithdrawRequest
	if err := ctx.Bind(&request); err != nil || request.CurrentPassword == "" {
		return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "現在のパスワードを入力してください")
	}
	result, err := c.userUsecase.Withdraw(ctx.Request().Context(), userID, usecase.WithdrawInput{CurrentPassword: request.CurrentPassword})
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrCurrentPasswordMismatch):
			return writeAPIError(ctx, http.StatusUnauthorized, "CURRENT_PASSWORD_MISMATCH", "現在のパスワードが正しくありません")
		case errors.Is(err, entity.ErrInvalidCredentials), errors.Is(err, entity.ErrUserNotFound):
			return writeAPIError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です")
		case errors.Is(err, entity.ErrInvalidInput):
			return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "現在のパスワードを入力してください")
		default:
			return writeAPIError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "サーバー内部でエラーが発生しました")
		}
	}
	clearAuthCookies(ctx)
	return ctx.JSON(http.StatusOK, WithdrawResponse{Message: result.Message})
}

func (c *userController) UpdateProfile(ctx echo.Context) error {
	userID, ok := ctx.Get(middleware.AuthUserIDKey).(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return writeAPIError(ctx, http.StatusUnauthorized, "UNAUTHORIZED", "認証が必要です")
	}
	var request UpdateProfileRequest
	if err := ctx.Bind(&request); err != nil {
		return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "入力内容を確認してください")
	}
	user, err := c.userUsecase.UpdateProfile(ctx.Request().Context(), userID, usecase.UpdateProfileInput{Name: request.Name, Email: request.Email})
	if err != nil {
		switch {
		case errors.Is(err, entity.ErrEmailAlreadyExists):
			return writeAPIError(ctx, http.StatusConflict, "EMAIL_ALREADY_EXISTS", "このメールアドレスは既に登録されています")
		case errors.Is(err, entity.ErrInvalidInput):
			return writeAPIError(ctx, http.StatusBadRequest, "INVALID_INPUT", "入力内容を確認してください")
		case errors.Is(err, entity.ErrUserNotFound):
			return writeAPIError(ctx, http.StatusNotFound, "USER_NOT_FOUND", "ユーザーが見つかりません")
		default:
			return writeAPIError(ctx, http.StatusInternalServerError, "INTERNAL_ERROR", "サーバー内部でエラーが発生しました")
		}
	}
	return ctx.JSON(http.StatusOK, LoginUserResponse{ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role, Status: user.Status})
}

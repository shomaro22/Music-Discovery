package entity

import "errors"

var (
	ErrInvalidInput              = errors.New("invalid input")
	ErrInvalidUserForCreate      = errors.New("invalid user for create")
	ErrEmailAlreadyExists        = errors.New("email already exists")
	ErrUserNotFound              = errors.New("user not found")
	ErrInternal                  = errors.New("internal error")
	ErrInvalidCredentials        = errors.New("invalid credentials")
	ErrUserSuspended             = errors.New("user suspended")
	ErrInvalidRefreshToken       = errors.New("invalid refresh token")
	ErrRefreshTokenUsed          = errors.New("refresh token used")
	ErrRefreshTokenRevoked       = errors.New("refresh token revoked")
	ErrRefreshTokenExpired       = errors.New("refresh token expired")
	ErrRefreshTokenReuseDetected = errors.New("refresh token reuse detected")
)

type APIErrorResponse struct {
	Status    int    `json:"status"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

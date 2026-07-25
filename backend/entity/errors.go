package entity

import "errors"

var (
	ErrInvalidInput             = errors.New("invalid input")
	ErrInvalidUserForCreate     = errors.New("invalid user for create")
	ErrEmailAlreadyExists       = errors.New("email already exists")
	ErrUserNotFound             = errors.New("user not found")
	ErrUserAlreadyWithdrawn     = errors.New("user already withdrawn")
	ErrInternal                 = errors.New("internal error")
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrUserSuspended            = errors.New("user suspended")
	ErrInvalidRefreshToken      = errors.New("invalid refresh token")
	ErrRefreshTokenReused       = errors.New("refresh token reused")
	ErrCurrentPasswordMismatch  = errors.New("current password mismatch")
	ErrInvalidRoleConfiguration = errors.New("invalid role configuration")
	ErrForbidden                = errors.New("forbidden")
	ErrNotFound                 = errors.New("not found")
	ErrConflict                 = errors.New("conflict")
	ErrAlreadyActive            = errors.New("user already active")
	ErrAlreadySuspended         = errors.New("user already suspended")
	ErrLastSystemAdmin          = errors.New("last active system admin")
	ErrReasonRequired           = errors.New("reason required")
)

type APIErrorResponse struct {
	Status    int    `json:"status"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

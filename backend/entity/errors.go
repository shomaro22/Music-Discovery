package entity

import "errors"

var (
	ErrInvalidInput         = errors.New("invalid input")
	ErrInvalidUserForCreate = errors.New("invalid user for create")
	ErrEmailAlreadyExists   = errors.New("email already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrInternal             = errors.New("internal error")
)

type APIErrorResponse struct {
	Status    int    `json:"status"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

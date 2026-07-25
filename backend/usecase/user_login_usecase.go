package usecase

import (
	"context"
	"errors"

	"music-discovery/backend/entity"
)

func (u *userUsecase) Login(ctx context.Context, input LoginInput) (*LoginResult, error) {
	values, validationErrors := u.userValidator.ValidateLogin(input.Email, input.Password)
	if len(validationErrors) > 0 {
		return nil, entity.ErrInvalidInput
	}
	input = LoginInput{Email: values.Email, Password: values.Password}
	user, err := u.userRepository.FindByEmail(ctx, input.Email)
	if errors.Is(err, entity.ErrUserNotFound) {
		_ = u.tokenService.ComparePassword(dummyPasswordHash, input.Password)
		return nil, entity.ErrInvalidCredentials
	}
	if err != nil {
		return nil, entity.ErrInternal
	}
	if err := u.tokenService.ComparePassword(user.PasswordHash, input.Password); err != nil {
		return nil, entity.ErrInvalidCredentials
	}
	if !user.Role.IsPersistable() {
		return nil, entity.ErrInvalidCredentials
	}
	if user.Status == entity.UserStatusSuspended {
		return nil, entity.ErrUserSuspended
	}
	if user.Status != entity.UserStatusActive {
		return nil, entity.ErrInvalidCredentials
	}
	result, err := u.issueAuthResult(ctx, user, u.clock.Now())
	if err != nil {
		return nil, err
	}
	return (*LoginResult)(result), nil
}

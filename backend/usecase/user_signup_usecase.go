package usecase

import (
	"context"
	"errors"
	"time"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
)

func (u *userUsecase) SignUp(ctx context.Context, input SignUpInput) (*SignUpResult, error) {
	values, validationErrors := u.userValidator.ValidateSignUp(input.Name, input.Email, input.Password)
	if len(validationErrors) > 0 {
		return nil, entity.ErrInvalidInput
	}
	input = SignUpInput{Name: values.Name, Email: values.Email, Password: values.Password}
	_, err := u.userRepository.FindByEmail(ctx, input.Email)
	switch {
	case err == nil:
		return nil, entity.ErrEmailAlreadyExists
	case errors.Is(err, entity.ErrUserNotFound):
	default:
		return nil, entity.ErrInternal
	}
	passwordHash, err := u.passwordHasher.Hash(input.Password)
	if err != nil {
		return nil, entity.ErrInternal
	}
	now := u.clock.Now()
	user := &entity.User{
		ID: uuid.New(), Name: input.Name, Email: input.Email, PasswordHash: passwordHash,
		Role: entity.UserRoleUser, Status: entity.UserStatusActive, TokenVersion: 0,
		CreatedAt: now, UpdatedAt: now,
	}
	authResult, refreshToken, err := u.buildAuthResult(user, now)
	if err != nil {
		return nil, err
	}
	if err := u.authTransactionManager.CreateUserAndRefreshToken(ctx, user, refreshToken); err != nil {
		switch {
		case errors.Is(err, entity.ErrEmailAlreadyExists):
			return nil, entity.ErrEmailAlreadyExists
		default:
			return nil, entity.ErrInternal
		}
	}

	return authResult, nil
}

func (u *userUsecase) buildAuthResult(user *entity.User, now time.Time) (*AuthResult, *entity.RefreshToken, error) {
	access, expiresIn, err := u.tokenService.GenerateAccessToken(user, now)
	if err != nil {
		return nil, nil, entity.ErrInternal
	}
	plain, hash, err := u.tokenService.GenerateRefreshToken()
	if err != nil {
		return nil, nil, entity.ErrInternal
	}
	csrf, err := u.tokenService.GenerateCSRFToken()
	if err != nil {
		return nil, nil, entity.ErrInternal
	}
	expiresAt := now.Add(u.tokenService.RefreshTokenTTL())
	refresh := &entity.RefreshToken{
		ID: uuid.New(), UserID: user.ID, TokenHash: hash, FamilyID: u.tokenService.GenerateFamilyID(),
		TokenVersion: user.TokenVersion, ExpiresAt: expiresAt, CreatedAt: now, UpdatedAt: now,
	}
	result := &AuthResult{
		AccessToken: access, ExpiresIn: expiresIn, RefreshToken: plain,
		RefreshTokenExpiresAt: expiresAt, CSRFToken: csrf, User: user,
	}
	return result, refresh, nil
}

func (u *userUsecase) issueAuthResult(ctx context.Context, user *entity.User, now time.Time) (*AuthResult, error) {
	result, refresh, err := u.buildAuthResult(user, now)
	if err != nil {
		return nil, err
	}
	if err := u.refreshTokenRepository.Create(ctx, refresh); err != nil {
		return nil, entity.ErrInternal
	}
	return result, nil
}

const dummyPasswordHash = "$2a$10$7EqJtq98hPqEX7fNZaFWoO5LQzYOgTBz8tLJ7YQnbvCw8wq9a3E0K"

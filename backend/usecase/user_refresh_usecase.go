package usecase

import (
	"context"
	"errors"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
)

func (u *userUsecase) Refresh(ctx context.Context, plainToken string) (*RefreshResult, error) {
	if plainToken == "" {
		return nil, entity.ErrInvalidRefreshToken
	}
	now := u.clock.Now()
	current, err := u.refreshTokenRepository.FindByTokenHash(ctx, u.tokenService.HashRefreshToken(plainToken))
	if err != nil {
		if errors.Is(err, entity.ErrInvalidRefreshToken) {
			return nil, entity.ErrInvalidRefreshToken
		}
		return nil, entity.ErrInternal
	}

	if current.IsRevoked() || current.IsUsed() {
		if err := u.authTransactionManager.RevokeFamilyAndIncrementTokenVersion(ctx, current.FamilyID, current.UserID, now); err != nil {
			if errors.Is(err, entity.ErrInvalidRefreshToken) {
				return nil, entity.ErrInvalidRefreshToken
			}
			return nil, entity.ErrInternal
		}
		return nil, entity.ErrRefreshTokenReused
	}
	if current.IsExpired(now) {
		return nil, entity.ErrInvalidRefreshToken
	}
	user, err := u.userRepository.FindByID(ctx, current.UserID)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			return nil, entity.ErrInvalidRefreshToken
		}
		return nil, entity.ErrInternal
	}
	if current.TokenVersion != user.TokenVersion {
		return nil, entity.ErrInvalidRefreshToken
	}
	if user.Status == entity.UserStatusSuspended {
		return nil, entity.ErrUserSuspended
	}
	if user.Status != entity.UserStatusActive {
		return nil, entity.ErrInvalidRefreshToken
	}
	newPlain, newHash, err := u.tokenService.GenerateRefreshToken()
	if err != nil {
		return nil, entity.ErrInternal
	}
	expiresAt := now.Add(u.tokenService.RefreshTokenTTL())
	next := &entity.RefreshToken{
		ID: uuid.New(), UserID: user.ID, TokenHash: newHash, FamilyID: current.FamilyID,
		TokenVersion: user.TokenVersion, ExpiresAt: expiresAt, CreatedAt: now, UpdatedAt: now,
	}
	if err := u.refreshTokenRepository.Rotate(ctx, current.ID, next, now); err != nil {
		if errors.Is(err, entity.ErrRefreshTokenReused) {
			if revokeErr := u.authTransactionManager.RevokeFamilyAndIncrementTokenVersion(ctx, current.FamilyID, current.UserID, now); revokeErr != nil {
				if errors.Is(revokeErr, entity.ErrInvalidRefreshToken) {
					return nil, entity.ErrInvalidRefreshToken
				}
				return nil, entity.ErrInternal
			}
			return nil, entity.ErrRefreshTokenReused
		}
		return nil, entity.ErrInternal
	}
	accessToken, expiresIn, err := u.tokenService.GenerateAccessToken(user, now)
	if err != nil {
		return nil, entity.ErrInternal
	}
	csrfToken, err := u.tokenService.GenerateCSRFToken()
	if err != nil {
		return nil, entity.ErrInternal
	}
	return &RefreshResult{
		AccessToken: accessToken, ExpiresIn: expiresIn, RefreshToken: newPlain,
		RefreshTokenExpiresAt: expiresAt, CSRFToken: csrfToken,
	}, nil
}

func (u *userUsecase) Logout(ctx context.Context, plainToken string) error {
	if plainToken == "" {
		return nil
	}
	token, err := u.refreshTokenRepository.FindByTokenHash(ctx, u.tokenService.HashRefreshToken(plainToken))
	if err != nil {
		if errors.Is(err, entity.ErrInvalidRefreshToken) {
			return nil
		}
		return entity.ErrInternal
	}
	if token == nil || token.FamilyID == uuid.Nil {
		return nil
	}
	if err := u.authTransactionManager.RevokeFamilyAndIncrementTokenVersion(ctx, token.FamilyID, token.UserID, u.clock.Now()); err != nil {
		if errors.Is(err, entity.ErrInvalidRefreshToken) {
			// Logout is idempotent: an already-revoked family is treated as success.
			return nil
		}
		return entity.ErrInternal
	}
	return nil
}

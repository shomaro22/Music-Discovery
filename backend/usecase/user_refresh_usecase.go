package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
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
	// Fast pre-check avoids unnecessary rotation work for an already consumed token.
	// Rotate performs the authoritative row-locked check to close concurrent reuse races.
	if current.IsRevoked() || current.IsUsed() {
		if err := u.authTransactionManager.RevokeFamilyAndIncrementTokenVersion(ctx, current.FamilyID, current.UserID, now); err != nil {
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
	if err := u.refreshTokenRepository.RevokeFamily(ctx, token.FamilyID, u.clock.Now()); err != nil {
		return entity.ErrInternal
	}
	return nil
}

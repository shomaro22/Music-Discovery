package usecase

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
	"music-discovery/backend/repository"
)

func (u *userUsecase) CurrentUser(ctx context.Context, userID uuid.UUID) (*CurrentUserResult, error) {
	if userID == uuid.Nil {
		return nil, entity.ErrInvalidCredentials
	}

	user, err := u.userRepository.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			return nil, entity.ErrInvalidCredentials
		}
		return nil, entity.ErrInternal
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

	return &CurrentUserResult{
		ID: user.ID, Name: user.Name, Email: user.Email, Role: user.Role, Status: user.Status,
	}, nil
}

func (u *userUsecase) ValidateTokenVersion(ctx context.Context, userID uuid.UUID, tokenVersion uint64) (*entity.User, error) {
	user, err := u.userRepository.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !user.Role.IsPersistable() {
		return nil, entity.ErrInvalidCredentials
	}
	if user.Status == entity.UserStatusSuspended {
		return nil, entity.ErrUserSuspended
	}
	if user.Status != entity.UserStatusActive || user.TokenVersion != tokenVersion {
		return nil, entity.ErrInvalidCredentials
	}
	return user, nil
}

func (u *userUsecase) Withdraw(ctx context.Context, userID uuid.UUID, input WithdrawInput) (*WithdrawResult, error) {
	if userID == uuid.Nil || input.CurrentPassword == "" {
		return nil, entity.ErrInvalidInput
	}
	user, err := u.userRepository.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			return nil, entity.ErrInvalidCredentials
		}
		return nil, entity.ErrInternal
	}
	if user.Status != entity.UserStatusActive {
		return nil, entity.ErrInvalidCredentials
	}
	if err := u.tokenService.ComparePassword(user.PasswordHash, input.CurrentPassword); err != nil {
		return nil, entity.ErrCurrentPasswordMismatch
	}

	now := u.clock.Now()
	if err := u.authTransactionManager.WithinTransaction(
		ctx,
		func(tx repository.ITransaction) error {
			if err := u.userRepository.SoftDelete(ctx, tx, userID, now); err != nil {
				return err
			}
			if err := u.userRepository.IncrementTokenVersion(ctx, tx, userID, now); err != nil {
				return err
			}
			if err := u.refreshTokenRepository.RevokeAllByUserID(ctx, tx, userID, now); err != nil {
				return err
			}
			return nil
		}); err != nil {
		switch {
		case errors.Is(err, entity.ErrUserNotFound),
			errors.Is(err, entity.ErrUserAlreadyWithdrawn):
			return nil, entity.ErrInvalidCredentials
		default:
			return nil, entity.ErrInternal
		}
	}
	return &WithdrawResult{Message: "退会が完了しました。"}, nil
}

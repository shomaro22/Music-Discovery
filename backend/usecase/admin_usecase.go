package usecase

import (
	"context"
	"errors"
	"time"

	"music-discovery/backend/entity"
	"music-discovery/backend/repository"

	"github.com/google/uuid"
)

type ChangeRoleInput struct {
	ActorUserID  uuid.UUID
	TargetUserID uuid.UUID
	Role         entity.Role
}

type IAdminUsecase interface {
	ChangeRole(ctx context.Context, input ChangeRoleInput) error
}

type adminUsecase struct {
	userRepository         repository.IUserRepository
	roleTransactionManager repository.IRoleTransactionManager
	clock                  IClock
}

func NewAdminUsecase(userRepository repository.IUserRepository, roleTransactionManager repository.IRoleTransactionManager, clock IClock) IAdminUsecase {
	return &adminUsecase{userRepository: userRepository, roleTransactionManager: roleTransactionManager, clock: clock}
}

func (u *adminUsecase) ChangeRole(ctx context.Context, input ChangeRoleInput) error {
	if input.ActorUserID == uuid.Nil || input.TargetUserID == uuid.Nil || !input.Role.IsPersistable() {
		return entity.ErrInvalidInput
	}
	actor, err := u.userRepository.FindByID(ctx, input.ActorUserID)
	if err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			return entity.ErrInvalidCredentials
		}
		return entity.ErrInternal
	}
	if actor.Status != entity.UserStatusActive || actor.Role != entity.RoleSystemAdmin {
		return entity.ErrForbidden
	}
	if err := u.roleTransactionManager.ChangeUserRole(ctx, input.TargetUserID, input.Role, u.clock.Now()); err != nil {
		if errors.Is(err, entity.ErrUserNotFound) {
			return entity.ErrUserNotFound
		}
		return entity.ErrInternal
	}
	return nil
}

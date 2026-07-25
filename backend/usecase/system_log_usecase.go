package usecase

import (
	"context"

	"music-discovery/backend/entity"
)

type ISystemLogRepository interface {
	CreateSystemLog(context.Context, *entity.SystemLog) error
}

type ISystemLogUsecase interface {
	CreateSystemLog(context.Context, *entity.SystemLog) error
}

type systemLogUsecase struct {
	repository ISystemLogRepository
}

func NewSystemLogUsecase(repository ISystemLogRepository) ISystemLogUsecase {
	return &systemLogUsecase{repository: repository}
}

func (u *systemLogUsecase) CreateSystemLog(ctx context.Context, log *entity.SystemLog) error {
	if u == nil || u.repository == nil || log == nil {
		return entity.ErrInvalidInput
	}
	return u.repository.CreateSystemLog(ctx, log)
}

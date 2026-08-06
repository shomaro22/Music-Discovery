package usecase

import (
	"context"
	"music-discovery/backend/repository"
	"time"
)

type LogRetentionConfig struct {
	SystemDays    int
	SensitiveDays int
	AuditDays     int
}
type LogRetentionResult struct {
	SystemDeleted int64
	AuditDeleted  int64
}
type ILogRetentionUsecase interface {
	Cleanup(context.Context) (LogRetentionResult, error)
}
type logRetentionUsecase struct {
	repository repository.ILogRetentionRepository
	config     LogRetentionConfig
}

func NewLogRetentionUsecase(repository repository.ILogRetentionRepository, config LogRetentionConfig) ILogRetentionUsecase {
	if config.SystemDays <= 0 {
		config.SystemDays = 90
	}
	if config.SensitiveDays <= 0 {
		config.SensitiveDays = 180
	}
	if config.AuditDays <= 0 {
		config.AuditDays = 365
	}
	return &logRetentionUsecase{repository: repository, config: config}
}
func (u *logRetentionUsecase) Cleanup(ctx context.Context) (LogRetentionResult, error) {
	now := time.Now().UTC()
	system, err := u.repository.DeleteExpiredSystemLogs(ctx, now.AddDate(0, 0, -u.config.SystemDays), now.AddDate(0, 0, -u.config.SensitiveDays))
	if err != nil {
		return LogRetentionResult{}, err
	}
	audit, err := u.repository.DeleteExpiredAuditLogs(ctx, now.AddDate(0, 0, -u.config.AuditDays))
	return LogRetentionResult{SystemDeleted: system, AuditDeleted: audit}, err
}

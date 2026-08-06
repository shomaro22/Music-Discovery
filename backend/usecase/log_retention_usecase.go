package usecase

import (
	"context"
	"time"
)

type LogRetentionPort interface {
	DeleteExpiredSystemLogs(context.Context, time.Time, time.Time) (int64, error)
	DeleteExpiredAuditLogs(context.Context, time.Time) (int64, error)
}

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
	port   LogRetentionPort
	config LogRetentionConfig
	now    func() time.Time
}

func NewLogRetentionUsecase(port LogRetentionPort, config LogRetentionConfig) ILogRetentionUsecase {
	return newLogRetentionUsecase(port, config, func() time.Time { return time.Now().UTC() })
}

func newLogRetentionUsecase(port LogRetentionPort, config LogRetentionConfig, now func() time.Time) ILogRetentionUsecase {
	if config.SystemDays <= 0 {
		config.SystemDays = 90
	}
	if config.SensitiveDays <= 0 {
		config.SensitiveDays = 180
	}
	if config.AuditDays <= 0 {
		config.AuditDays = 365
	}
	return &logRetentionUsecase{port: port, config: config, now: now}
}
func (u *logRetentionUsecase) Cleanup(ctx context.Context) (LogRetentionResult, error) {
	now := u.now()
	system, err := u.port.DeleteExpiredSystemLogs(ctx, now.AddDate(0, 0, -u.config.SystemDays), now.AddDate(0, 0, -u.config.SensitiveDays))
	if err != nil {
		return LogRetentionResult{}, err
	}
	audit, err := u.port.DeleteExpiredAuditLogs(ctx, now.AddDate(0, 0, -u.config.AuditDays))
	return LogRetentionResult{SystemDeleted: system, AuditDeleted: audit}, err
}

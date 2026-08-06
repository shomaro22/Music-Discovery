package usecase

import (
	"context"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
)

func (u *adminManagementUsecase) ListAuditLogs(ctx context.Context, in PageInput) (PageResult[AuditLogListResponse], error) {
	q, e := query(in)
	if e != nil {
		return PageResult[AuditLogListResponse]{}, e
	}
	items, total, e := u.repo.ListAuditLogs(ctx, q)
	if e != nil {
		return PageResult[AuditLogListResponse]{}, e
	}
	values := make([]AuditLogListResponse, 0, len(items))
	for _, item := range items {
		values = append(values, AuditLogListResponse{ID: item.ID, ActorUserID: item.ActorUserID, ActorRole: item.ActorRole, ActorName: item.ActorName, ActorEmail: item.ActorEmail, RequestID: item.RequestID, Action: item.Action, TargetType: item.TargetType, TargetID: item.TargetID, Summary: item.Summary, IPAddress: item.IPAddress, UserAgent: item.UserAgent, CreatedAt: item.CreatedAt})
	}
	return page(values, total, in)
}
func (u *adminManagementUsecase) GetAuditLog(ctx context.Context, id uuid.UUID) (*entity.AuditLog, error) {
	if id == uuid.Nil {
		return nil, entity.ErrInvalidInput
	}
	return u.repo.FindAuditLog(ctx, id)
}
func (u *adminManagementUsecase) ListSystemLogs(ctx context.Context, in PageInput) (PageResult[SystemLogListResponse], error) {
	q, e := query(in)
	if e != nil {
		return PageResult[SystemLogListResponse]{}, e
	}
	items, total, e := u.repo.ListSystemLogs(ctx, q)
	if e != nil {
		return PageResult[SystemLogListResponse]{}, e
	}
	values := make([]SystemLogListResponse, 0, len(items))
	for _, item := range items {
		values = append(values, SystemLogListResponse{ID: item.ID, Environment: item.Environment, Service: item.Service, Version: item.Version, Level: item.Level, Category: item.Category, Event: item.Event, ErrorCode: item.ErrorCode, Message: item.Message, RequestID: item.RequestID, UserID: item.UserID, Method: item.Method, Path: item.Path, StatusCode: item.StatusCode, ExternalService: item.ExternalService, DurationMs: item.DurationMs, IPAddress: item.IPAddress, UserAgent: item.UserAgent, CreatedAt: item.CreatedAt})
	}
	return page(values, total, in)
}
func (u *adminManagementUsecase) GetSystemLog(ctx context.Context, id uuid.UUID) (*entity.SystemLog, error) {
	if id == uuid.Nil {
		return nil, entity.ErrInvalidInput
	}
	return u.repo.FindSystemLog(ctx, id)
}

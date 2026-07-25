package usecase

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"music-discovery/backend/entity"
)

func (u *adminManagementUsecase) ListUsers(ctx context.Context, in PageInput) (PageResult[UserAdminResponse], error) {
	if !validateSort(in.Sort, "name", "email", "role", "status", "created_at", "updated_at", "deleted_at") {
		return PageResult[UserAdminResponse]{}, entity.ErrInvalidInput
	}
	q, e := query(in)
	if e != nil {
		return PageResult[UserAdminResponse]{}, e
	}
	items, total, e := u.repo.ListUsers(ctx, q)
	if e != nil {
		return PageResult[UserAdminResponse]{}, e
	}
	responses := make([]UserAdminResponse, len(items))
	for i := range items {
		responses[i] = newUserAdminResponse(items[i])
	}
	return page(responses, total, in)
}
func (u *adminManagementUsecase) GetUser(ctx context.Context, id uuid.UUID) (*UserDetailResponse, error) {
	if id == uuid.Nil {
		return nil, entity.ErrInvalidInput
	}
	value, e := u.repo.FindUserUnscoped(ctx, id)
	if e != nil {
		return nil, e
	}
	metrics, e := u.repo.FindUserMetrics(ctx, id)
	if e != nil {
		return nil, e
	}
	return &UserDetailResponse{UserAdminResponse: newUserAdminResponse(*value), ProfileStatus: metrics.ProfileStatus, RecommendationCount: metrics.RecommendationCount, FeedbackCount: metrics.FeedbackCount, LastLoginAt: metrics.LastLoginAt}, nil
}
func (u *adminManagementUsecase) SetUserStatus(ctx context.Context, actor, id uuid.UUID, in UserStatusInput, meta AuditInput) error {
	reason := strings.TrimSpace(in.Reason)
	if actor == uuid.Nil || id == uuid.Nil || (in.Status != entity.UserStatusActive && in.Status != entity.UserStatusSuspended) {
		return entity.ErrInvalidInput
	}
	if in.Status == entity.UserStatusSuspended && reason == "" {
		return entity.ErrReasonRequired
	}
	if actor == id && in.Status == entity.UserStatusSuspended {
		return entity.ErrForbidden
	}
	action := "user_activated"
	if in.Status == entity.UserStatusSuspended {
		action = "user_suspended"
	}
	return u.repo.SetUserStatusWithAudit(ctx, actor, id, in.Status, reason, u.clock.Now(), u.audit(actor, action, "user", reason, "User状態を変更しました", meta))
}
func (u *adminManagementUsecase) ListRoles(context.Context) []entity.Role {
	return []entity.Role{entity.RoleUser, entity.RoleContentAdmin, entity.RoleSystemAdmin}
}
func (u *adminManagementUsecase) ChangeRole(ctx context.Context, actor, id uuid.UUID, in RoleChangeInput, meta AuditInput) (*RoleChangeResponse, error) {
	reason := strings.TrimSpace(in.Reason)
	if actor == uuid.Nil || id == uuid.Nil || !in.Role.IsPersistable() {
		return nil, entity.ErrInvalidInput
	}
	if reason == "" {
		return nil, entity.ErrReasonRequired
	}
	if actor == id {
		return nil, entity.ErrForbidden
	}
	target, e := u.repo.FindUserUnscoped(ctx, id)
	if e != nil {
		return nil, e
	}
	before := target.Role
	if e = u.repo.ChangeUserRoleWithAudit(ctx, id, in.Role, u.clock.Now(), u.audit(actor, "user_role_changed", "user", reason, "User Roleを変更しました", meta)); e != nil {
		return nil, e
	}
	return &RoleChangeResponse{UserID: id, BeforeRole: before, AfterRole: in.Role}, nil
}

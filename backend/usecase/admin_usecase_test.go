package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"music-discovery/backend/entity"

	"github.com/google/uuid"
)

type roleTransactionManagerStub struct {
	calls  int
	target uuid.UUID
	role   entity.Role
	at     time.Time
	err    error
}

func (s *roleTransactionManagerStub) ChangeUserRole(_ context.Context, target uuid.UUID, role entity.Role, at time.Time) error {
	s.calls++
	s.target = target
	s.role = role
	s.at = at
	return s.err
}

func TestAdminUsecaseChangeRole(t *testing.T) {
	actorID, targetID := uuid.New(), uuid.New()
	now := time.Date(2026, 7, 23, 3, 0, 0, 0, time.UTC)
	repo := &fakeUserRepository{findUser: &entity.User{ID: actorID, Role: entity.RoleSystemAdmin, Status: entity.UserStatusActive}}
	tx := &roleTransactionManagerStub{}
	err := NewAdminUsecase(repo, tx, &fakeClock{now: now}).ChangeRole(context.Background(), ChangeRoleInput{ActorUserID: actorID, TargetUserID: targetID, Role: entity.RoleContentAdmin})
	if err != nil || tx.calls != 1 || tx.target != targetID || tx.role != entity.RoleContentAdmin || !tx.at.Equal(now) {
		t.Fatalf("err=%v tx=%+v", err, tx)
	}
}

func TestAdminUsecaseRejectsUnauthorizedAndInvalidRole(t *testing.T) {
	actorID, targetID := uuid.New(), uuid.New()
	repo := &fakeUserRepository{findUser: &entity.User{ID: actorID, Role: entity.RoleUser, Status: entity.UserStatusActive}}
	tx := &roleTransactionManagerStub{}
	u := NewAdminUsecase(repo, tx, &fakeClock{now: time.Now()})
	if err := u.ChangeRole(context.Background(), ChangeRoleInput{ActorUserID: actorID, TargetUserID: targetID, Role: entity.RoleContentAdmin}); !errors.Is(err, entity.ErrForbidden) {
		t.Fatalf("err=%v", err)
	}
	if err := u.ChangeRole(context.Background(), ChangeRoleInput{ActorUserID: actorID, TargetUserID: targetID, Role: entity.RoleGuest}); !errors.Is(err, entity.ErrInvalidInput) {
		t.Fatalf("err=%v", err)
	}
	if tx.calls != 0 {
		t.Fatalf("calls=%d", tx.calls)
	}
}

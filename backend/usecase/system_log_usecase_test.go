package usecase

import (
	"context"
	"testing"

	"music-discovery/backend/entity"
)

type systemLogRepositoryStub struct {
	created *entity.SystemLog
}

func (s *systemLogRepositoryStub) CreateSystemLog(_ context.Context, log *entity.SystemLog) error {
	s.created = log
	return nil
}

func TestSystemLogUsecaseCreatesLogThroughRepositoryPort(t *testing.T) {
	repository := &systemLogRepositoryStub{}
	uc := NewSystemLogUsecase(repository)
	log := &entity.SystemLog{Event: "test_event"}
	if err := uc.CreateSystemLog(context.Background(), log); err != nil {
		t.Fatal(err)
	}
	if repository.created != log {
		t.Fatal("system log was not passed to repository port")
	}
}

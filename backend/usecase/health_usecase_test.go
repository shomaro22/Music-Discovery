package usecase

import (
	"context"
	"errors"
	"testing"
	"time"
)

type healthPortStub struct{ postgresErr, redisErr error }

func (s healthPortStub) PingPostgres(context.Context) error { return s.postgresErr }
func (s healthPortStub) PingRedis(context.Context) error    { return s.redisErr }

func TestHealthUsecaseReadiness(t *testing.T) {
	tests := []struct {
		name string
		port healthPortStub
		want string
	}{
		{"ready", healthPortStub{}, "ready"},
		{"postgres down", healthPortStub{postgresErr: errors.New("postgres://secret-host")}, "not_ready"},
		{"redis down", healthPortStub{redisErr: errors.New("redis secret")}, "not_ready"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := NewHealthUsecase(tt.port, "v1", time.Second)
			got := u.Readiness(context.Background())
			if got.Status != tt.want {
				t.Fatalf("status=%s", got.Status)
			}
			for _, dep := range got.Dependencies {
				if dep.Status == "down" && dep.ErrorCode != "CONNECTION_FAILED" {
					t.Fatalf("error_code=%s", dep.ErrorCode)
				}
			}
		})
	}
}

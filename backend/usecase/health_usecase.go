package usecase

import (
	"context"
	"time"

	"music-discovery/backend/repository"
)

type DependencyHealth struct {
	Status    string    `json:"status"`
	LatencyMs int64     `json:"latency_ms"`
	Error     string    `json:"error,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

type ReadinessResult struct {
	Status       string                      `json:"status"`
	Dependencies map[string]DependencyHealth `json:"dependencies"`
	Version      string                      `json:"version"`
	CheckedAt    time.Time                   `json:"checked_at"`
}

type IHealthUsecase interface {
	Readiness(context.Context) ReadinessResult
}

type healthUsecase struct {
	repository repository.IHealthCheckRepository
	version    string
	timeout    time.Duration
}

func NewHealthUsecase(repository repository.IHealthCheckRepository, version string, timeout time.Duration) IHealthUsecase {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return &healthUsecase{repository: repository, version: version, timeout: timeout}
}

func (u *healthUsecase) Readiness(ctx context.Context) ReadinessResult {
	result := ReadinessResult{Status: "ready", Version: u.version, CheckedAt: time.Now().UTC(), Dependencies: map[string]DependencyHealth{}}
	checks := []struct {
		name  string
		check func(context.Context) error
	}{
		{"postgres", u.repository.PingPostgres}, {"redis", u.repository.PingRedis},
	}
	for _, item := range checks {
		started := time.Now()
		checkCtx, cancel := context.WithTimeout(ctx, u.timeout)
		err := item.check(checkCtx)
		cancel()
		health := DependencyHealth{Status: "up", LatencyMs: time.Since(started).Milliseconds(), CheckedAt: time.Now().UTC()}
		if err != nil {
			health.Status = "down"
			health.Error = err.Error()
			result.Status = "not_ready"
		}
		result.Dependencies[item.name] = health
	}
	return result
}

package usecase

import (
	"context"
	"log/slog"
	"time"
)

type HealthCheckPort interface {
	PingPostgres(context.Context) error
	PingRedis(context.Context) error
}

type DependencyHealth struct {
	Status    string    `json:"status"`
	LatencyMs int64     `json:"latency_ms"`
	ErrorCode string    `json:"error_code,omitempty"`
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
	port    HealthCheckPort
	version string
	timeout time.Duration
	logger  *slog.Logger
}

func NewHealthUsecase(port HealthCheckPort, version string, timeout time.Duration, loggers ...*slog.Logger) IHealthUsecase {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	logger := slog.Default()
	if len(loggers) > 0 && loggers[0] != nil {
		logger = loggers[0]
	}
	return &healthUsecase{port: port, version: version, timeout: timeout, logger: logger}
}

func (u *healthUsecase) Readiness(ctx context.Context) ReadinessResult {
	result := ReadinessResult{Status: "ready", Version: u.version, CheckedAt: time.Now().UTC(), Dependencies: map[string]DependencyHealth{}}
	checks := []struct {
		name  string
		check func(context.Context) error
	}{
		{"postgres", u.port.PingPostgres},
		{"redis", u.port.PingRedis},
	}
	for _, item := range checks {
		started := time.Now()
		checkCtx, cancel := context.WithTimeout(ctx, u.timeout)
		err := item.check(checkCtx)
		cancel()
		health := DependencyHealth{Status: "up", LatencyMs: time.Since(started).Milliseconds(), CheckedAt: time.Now().UTC()}
		if err != nil {
			health.Status = "down"
			health.ErrorCode = "CONNECTION_FAILED"
			result.Status = "not_ready"
			u.logger.ErrorContext(ctx, "readiness dependency check failed", "dependency", item.name, "error", err)
		}
		result.Dependencies[item.name] = health
	}
	return result
}

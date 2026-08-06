package usecase

import (
	"context"
	"math"
	"runtime"
	"sort"
	"sync"
	"time"
)

const metricsSampleLimit = 4096

type MetricsSnapshot struct {
	RequestCount           uint64         `json:"request_count"`
	StatusCounts           map[int]uint64 `json:"status_counts"`
	ErrorCount             uint64         `json:"error_count"`
	ErrorRate              float64        `json:"error_rate"`
	ResponseTimeP50Ms      float64        `json:"response_time_p50_ms"`
	ResponseTimeP95Ms      float64        `json:"response_time_p95_ms"`
	ResponseTimeP99Ms      float64        `json:"response_time_p99_ms"`
	SlowRequestCount       uint64         `json:"slow_request_count"`
	SystemLogSaveFailures  uint64         `json:"system_log_save_failures"`
	LoginSuccesses         uint64         `json:"login_successes"`
	LoginFailures          uint64         `json:"login_failures"`
	RecommendationCount    uint64         `json:"recommendation_count"`
	RecommendationFailures uint64         `json:"recommendation_failures"`
	FallbackCount          uint64         `json:"fallback_count"`
	FeedbackCount          uint64         `json:"feedback_count"`
	DailyDiscoveryCount    uint64         `json:"daily_discovery_count"`
	DatabaseErrors         uint64         `json:"database_errors"`
	SpotifyAPIErrors       uint64         `json:"spotify_api_errors"`
	RateLimitExceeded      uint64         `json:"rate_limit_exceeded"`
}

type RuntimeMetrics struct {
	Goroutines     int    `json:"goroutines"`
	HeapAllocBytes uint64 `json:"heap_alloc_bytes"`
	HeapInuseBytes uint64 `json:"heap_inuse_bytes"`
	SystemBytes    uint64 `json:"system_bytes"`
}
type DatabaseMetrics struct {
	OpenConnections  int   `json:"open_connections"`
	InUseConnections int   `json:"in_use_connections"`
	IdleConnections  int   `json:"idle_connections"`
	WaitCount        int64 `json:"wait_count"`
}
type MonitoringMetrics struct {
	Application MetricsSnapshot `json:"application"`
	Runtime     RuntimeMetrics  `json:"runtime"`
	Database    DatabaseMetrics `json:"database"`
	Version     string          `json:"version"`
	GeneratedAt time.Time       `json:"generated_at"`
}

type IMetricsUsecase interface {
	RecordRequest(context.Context, int, time.Duration)
	Increment(context.Context, string)
	Snapshot(context.Context) MetricsSnapshot
	Monitoring(context.Context) MonitoringMetrics
	SetDatabaseStatsProvider(func() DatabaseMetrics)
}

type metricsUsecase struct {
	mu                           sync.RWMutex
	requestCount                 uint64
	statusCounts                 map[int]uint64
	errorCount, slowRequestCount uint64
	durationsMs                  []float64
	slowThreshold                time.Duration
	version                      string
	counters                     map[string]uint64
	databaseStats                func() DatabaseMetrics
}

func NewMetricsUsecase(slowThreshold time.Duration, version string) IMetricsUsecase {
	if slowThreshold <= 0 {
		slowThreshold = time.Second
	}
	return &metricsUsecase{statusCounts: map[int]uint64{}, durationsMs: make([]float64, 0, metricsSampleLimit), slowThreshold: slowThreshold, version: version, counters: map[string]uint64{}}
}
func (u *metricsUsecase) RecordRequest(_ context.Context, status int, duration time.Duration) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.requestCount++
	u.statusCounts[status]++
	if status >= 500 {
		u.errorCount++
	}
	if duration >= u.slowThreshold {
		u.slowRequestCount++
	}
	v := float64(duration.Microseconds()) / 1000
	if len(u.durationsMs) >= metricsSampleLimit {
		copy(u.durationsMs, u.durationsMs[1:])
		u.durationsMs[len(u.durationsMs)-1] = v
	} else {
		u.durationsMs = append(u.durationsMs, v)
	}
}
func (u *metricsUsecase) Increment(_ context.Context, name string) {
	u.mu.Lock()
	u.counters[name]++
	u.mu.Unlock()
}
func (u *metricsUsecase) SetDatabaseStatsProvider(provider func() DatabaseMetrics) {
	u.mu.Lock()
	u.databaseStats = provider
	u.mu.Unlock()
}
func (u *metricsUsecase) Snapshot(_ context.Context) MetricsSnapshot {
	u.mu.RLock()
	defer u.mu.RUnlock()
	statuses := map[int]uint64{}
	for k, v := range u.statusCounts {
		statuses[k] = v
	}
	values := append([]float64(nil), u.durationsMs...)
	sort.Float64s(values)
	r := MetricsSnapshot{RequestCount: u.requestCount, StatusCounts: statuses, ErrorCount: u.errorCount, SlowRequestCount: u.slowRequestCount, SystemLogSaveFailures: u.counters["system_log_save_failure"], LoginSuccesses: u.counters["login_success"], LoginFailures: u.counters["login_failure"], RecommendationCount: u.counters["recommendation_success"], RecommendationFailures: u.counters["recommendation_failure"], FallbackCount: u.counters["recommendation_fallback"], FeedbackCount: u.counters["feedback_saved"], DailyDiscoveryCount: u.counters["daily_discovery_success"], DatabaseErrors: u.counters["database_error"], SpotifyAPIErrors: u.counters["spotify_api_error"], RateLimitExceeded: u.counters["rate_limit_exceeded"]}
	if r.RequestCount > 0 {
		r.ErrorRate = float64(r.ErrorCount) / float64(r.RequestCount)
	}
	r.ResponseTimeP50Ms = percentile(values, .50)
	r.ResponseTimeP95Ms = percentile(values, .95)
	r.ResponseTimeP99Ms = percentile(values, .99)
	return r
}
func (u *metricsUsecase) Monitoring(ctx context.Context) MonitoringMetrics {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	u.mu.RLock()
	provider := u.databaseStats
	u.mu.RUnlock()
	db := DatabaseMetrics{}
	if provider != nil {
		db = provider()
	}
	return MonitoringMetrics{Application: u.Snapshot(ctx), Runtime: RuntimeMetrics{runtime.NumGoroutine(), mem.HeapAlloc, mem.HeapInuse, mem.Sys}, Database: db, Version: u.version, GeneratedAt: time.Now().UTC()}
}
func percentile(values []float64, ratio float64) float64 {
	if len(values) == 0 {
		return 0
	}
	index := int(math.Ceil(ratio*float64(len(values)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

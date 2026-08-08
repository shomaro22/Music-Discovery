package usecase

import (
	"context"
	"math"
	"runtime"
	runtimeMetrics "runtime/metrics"
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
	SpotifyAPIErrors       uint64         `json:"spotify_api_errors"`
	RateLimitExceeded      uint64         `json:"rate_limit_exceeded"`
}

type RuntimeMetrics struct {
	Goroutines            int     `json:"goroutines"`
	HeapAllocBytes        uint64  `json:"heap_alloc_bytes"`
	HeapInuseBytes        uint64  `json:"heap_inuse_bytes"`
	SystemBytes           uint64  `json:"system_bytes"`
	CPUUtilizationPercent float64 `json:"cpu_utilization_percent"`
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

type requestMetricEvent struct {
	at       time.Time
	status   int
	duration time.Duration
}
type counterMetricEvent struct {
	at   time.Time
	name string
}

type IMetricsUsecase interface {
	RecordRequest(context.Context, int, time.Duration)
	Increment(context.Context, string)
	Snapshot(context.Context) MetricsSnapshot
	WindowSnapshot(context.Context, time.Duration) MetricsSnapshot
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
	requestEvents                []requestMetricEvent
	counterEvents                []counterMetricEvent
	now                          func() time.Time
}

func NewMetricsUsecase(slowThreshold time.Duration, version string) IMetricsUsecase {
	if slowThreshold <= 0 {
		slowThreshold = time.Second
	}
	return &metricsUsecase{
		statusCounts: map[int]uint64{}, durationsMs: make([]float64, 0, metricsSampleLimit),
		slowThreshold: slowThreshold, version: version, counters: map[string]uint64{},
		requestEvents: make([]requestMetricEvent, 0, metricsSampleLimit), counterEvents: make([]counterMetricEvent, 0, metricsSampleLimit),
		now: time.Now,
	}
}
func (u *metricsUsecase) RecordRequest(_ context.Context, status int, duration time.Duration) {
	u.mu.Lock()
	defer u.mu.Unlock()
	now := u.now().UTC()
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
	u.requestEvents = appendRequestWindow(u.requestEvents, requestMetricEvent{at: now, status: status, duration: duration}, now.Add(-alertEvaluationWindow))
}
func (u *metricsUsecase) Increment(_ context.Context, name string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.counters[name]++
	now := u.now().UTC()
	u.counterEvents = appendCounterWindow(u.counterEvents, counterMetricEvent{at: now, name: name}, now.Add(-alertEvaluationWindow))
}
func appendRequestWindow(values []requestMetricEvent, value requestMetricEvent, cutoff time.Time) []requestMetricEvent {
	first := 0
	for first < len(values) && values[first].at.Before(cutoff) {
		first++
	}
	if first > 0 {
		values = append([]requestMetricEvent(nil), values[first:]...)
	}
	return append(values, value)
}
func appendCounterWindow(values []counterMetricEvent, value counterMetricEvent, cutoff time.Time) []counterMetricEvent {
	first := 0
	for first < len(values) && values[first].at.Before(cutoff) {
		first++
	}
	if first > 0 {
		values = append([]counterMetricEvent(nil), values[first:]...)
	}
	return append(values, value)
}
func (u *metricsUsecase) SetDatabaseStatsProvider(provider func() DatabaseMetrics) {
	u.mu.Lock()
	u.databaseStats = provider
	u.mu.Unlock()
}
func (u *metricsUsecase) Snapshot(_ context.Context) MetricsSnapshot {
	u.mu.RLock()
	defer u.mu.RUnlock()
	return snapshotFrom(u.requestCount, u.statusCounts, u.errorCount, u.slowRequestCount, u.durationsMs, u.counters)
}
func (u *metricsUsecase) WindowSnapshot(_ context.Context, window time.Duration) MetricsSnapshot {
	if window <= 0 {
		window = 5 * time.Minute
	}
	u.mu.RLock()
	defer u.mu.RUnlock()
	cutoff := u.now().UTC().Add(-window)
	statuses := map[int]uint64{}
	counters := map[string]uint64{}
	var requests, errors, slow uint64
	durations := make([]float64, 0)
	for _, event := range u.requestEvents {
		if event.at.Before(cutoff) {
			continue
		}
		requests++
		statuses[event.status]++
		if event.status >= 500 {
			errors++
		}
		if event.duration >= u.slowThreshold {
			slow++
		}
		durations = append(durations, float64(event.duration.Microseconds())/1000)
	}
	for _, event := range u.counterEvents {
		if !event.at.Before(cutoff) {
			counters[event.name]++
		}
	}
	return snapshotFrom(requests, statuses, errors, slow, durations, counters)
}
func snapshotFrom(requestCount uint64, statusCounts map[int]uint64, errorCount, slowCount uint64, durations []float64, counters map[string]uint64) MetricsSnapshot {
	statuses := map[int]uint64{}
	for k, v := range statusCounts {
		statuses[k] = v
	}
	values := append([]float64(nil), durations...)
	sort.Float64s(values)
	r := MetricsSnapshot{
		RequestCount: requestCount, StatusCounts: statuses, ErrorCount: errorCount, SlowRequestCount: slowCount,
		SystemLogSaveFailures: counters["system_log_save_failure"], LoginSuccesses: counters["login_success"], LoginFailures: counters["login_failure"],
		RecommendationCount: counters["recommendation_success"], RecommendationFailures: counters["recommendation_failure"], FallbackCount: counters["recommendation_fallback"],
		FeedbackCount: counters["feedback_saved"], DailyDiscoveryCount: counters["daily_discovery_success"],
		SpotifyAPIErrors: counters["spotify_api_error"], RateLimitExceeded: counters["rate_limit_exceeded"],
	}
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
	return MonitoringMetrics{
		Application: u.Snapshot(ctx),
		Runtime: RuntimeMetrics{
			Goroutines:            runtime.NumGoroutine(),
			HeapAllocBytes:        mem.HeapAlloc,
			HeapInuseBytes:        mem.HeapInuse,
			SystemBytes:           mem.Sys,
			CPUUtilizationPercent: runtimeCPUUtilizationPercent(),
		},
		Database:    db,
		Version:     u.version,
		GeneratedAt: time.Now().UTC(),
	}
}

func runtimeCPUUtilizationPercent() float64 {
	samples := []runtimeMetrics.Sample{
		{Name: "/cpu/classes/total:cpu-seconds"},
		{Name: "/cpu/classes/idle:cpu-seconds"},
	}
	runtimeMetrics.Read(samples)
	if samples[0].Value.Kind() != runtimeMetrics.KindFloat64 || samples[1].Value.Kind() != runtimeMetrics.KindFloat64 {
		return 0
	}
	total := samples[0].Value.Float64()
	idle := samples[1].Value.Float64()
	if total <= 0 {
		return 0
	}
	value := (total - idle) / total * 100
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
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

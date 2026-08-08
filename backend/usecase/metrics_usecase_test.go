package usecase

import (
	"context"
	"testing"
	"time"
)

func TestMetricsSnapshot(t *testing.T) {
	u := NewMetricsUsecase(100*time.Millisecond, "test")
	u.RecordRequest(context.Background(), 200, 10*time.Millisecond)
	u.RecordRequest(context.Background(), 500, 200*time.Millisecond)
	s := u.Snapshot(context.Background())
	if s.RequestCount != 2 || s.ErrorCount != 1 || s.SlowRequestCount != 1 {
		t.Fatalf("unexpected snapshot: %+v", s)
	}
	if s.ResponseTimeP95Ms != 200 || s.ResponseTimeP99Ms != 200 {
		t.Fatalf("nearest-rank percentile must select 200: p95=%v p99=%v", s.ResponseTimeP95Ms, s.ResponseTimeP99Ms)
	}
}
func TestPercentileNearestRankTwoValues(t *testing.T) {
	values := []float64{10, 200}
	if got := percentile(values, .95); got != 200 {
		t.Fatalf("p95=%v", got)
	}
	if got := percentile(values, .99); got != 200 {
		t.Fatalf("p99=%v", got)
	}
}

func TestRuntimeCPUUtilizationPercentIsBounded(t *testing.T) {
	value := runtimeCPUUtilizationPercent()
	if value < 0 || value > 100 {
		t.Fatalf("cpu utilization=%v want 0..100", value)
	}
}

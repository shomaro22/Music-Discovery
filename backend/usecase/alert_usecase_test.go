package usecase

import (
	"context"
	"testing"
	"time"
)

func TestAlertUsesFiveMinuteWindowAndRecovers(t *testing.T) {
	u := NewMetricsUsecase(time.Second, "v1").(*metricsUsecase)
	now := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)
	u.now = func() time.Time { return now }
	alert := NewAlertUsecase(u, .05, 1000)
	u.RecordRequest(context.Background(), 500, 10*time.Millisecond)
	states := alert.Evaluate(context.Background(), ReadinessResult{Status: "ready"})
	if states[0].Status != "firing" {
		t.Fatalf("expected firing: %+v", states[0])
	}
	now = now.Add(6 * time.Minute)
	states = alert.Evaluate(context.Background(), ReadinessResult{Status: "ready"})
	if states[0].Status != "resolved" {
		t.Fatalf("expected resolved: %+v", states[0])
	}
}

func TestAlertThresholdBoundaryDoesNotFire(t *testing.T) {
	u := NewMetricsUsecase(time.Second, "v1")
	for i := 0; i < 19; i++ {
		u.RecordRequest(context.Background(), 200, time.Millisecond)
	}
	u.RecordRequest(context.Background(), 500, time.Millisecond)
	states := NewAlertUsecase(u, .05, 1000).Evaluate(context.Background(), ReadinessResult{Status: "ready"})
	if states[0].Status != "resolved" {
		t.Fatalf("5 percent boundary must not fire")
	}
}

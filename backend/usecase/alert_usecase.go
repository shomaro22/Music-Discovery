package usecase

import (
	"context"
	"time"
)

const alertEvaluationWindow = 5 * time.Minute

type AlertState struct {
	Code      string    `json:"code"`
	Severity  string    `json:"severity"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	CheckedAt time.Time `json:"checked_at"`
}
type IAlertUsecase interface {
	Evaluate(context.Context, ReadinessResult) []AlertState
}
type alertUsecase struct {
	metrics                          IMetricsUsecase
	errorRateThreshold, p95Threshold float64
}

func NewAlertUsecase(metrics IMetricsUsecase, errorRateThreshold, p95Threshold float64) IAlertUsecase {
	if errorRateThreshold <= 0 {
		errorRateThreshold = .05
	}
	if p95Threshold <= 0 {
		p95Threshold = 1000
	}
	return &alertUsecase{metrics: metrics, errorRateThreshold: errorRateThreshold, p95Threshold: p95Threshold}
}
func (u *alertUsecase) Evaluate(ctx context.Context, readiness ReadinessResult) []AlertState {
	now := time.Now().UTC()
	s := u.metrics.WindowSnapshot(ctx, alertEvaluationWindow)
	alerts := []AlertState{
		{Code: "ERROR_RATE_HIGH", Severity: "critical", Status: statusFor(s.RequestCount > 0 && s.ErrorRate > u.errorRateThreshold), Message: "API error rate in the last 5 minutes exceeds threshold", Value: s.ErrorRate, Threshold: u.errorRateThreshold, CheckedAt: now},
		{Code: "RESPONSE_P95_HIGH", Severity: "warning", Status: statusFor(s.RequestCount > 0 && s.ResponseTimeP95Ms > u.p95Threshold), Message: "API p95 response time in the last 5 minutes exceeds threshold", Value: s.ResponseTimeP95Ms, Threshold: u.p95Threshold, CheckedAt: now},
		{Code: "SYSTEM_LOG_SAVE_FAILURE", Severity: "warning", Status: statusFor(s.SystemLogSaveFailures > 0), Message: "System log persistence failed in the last 5 minutes", Value: float64(s.SystemLogSaveFailures), Threshold: 0, CheckedAt: now},
	}
	if readiness.Status != "ready" {
		alerts = append(alerts, AlertState{Code: "DEPENDENCY_UNAVAILABLE", Severity: "critical", Status: "firing", Message: "Required dependency is unavailable", Value: 1, Threshold: 0, CheckedAt: now})
	} else {
		alerts = append(alerts, AlertState{Code: "DEPENDENCY_UNAVAILABLE", Severity: "critical", Status: "resolved", Message: "Required dependencies are available", Value: 0, Threshold: 0, CheckedAt: now})
	}
	return alerts
}
func statusFor(firing bool) string {
	if firing {
		return "firing"
	}
	return "resolved"
}

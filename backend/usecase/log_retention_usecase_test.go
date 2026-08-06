package usecase

import (
	"context"
	"testing"
	"time"
)

type retentionPortStub struct{ standard, sensitive, audit time.Time }

func (s *retentionPortStub) DeleteExpiredSystemLogs(_ context.Context, standard, sensitive time.Time) (int64, error) {
	s.standard = standard
	s.sensitive = sensitive
	return 1, nil
}
func (s *retentionPortStub) DeleteExpiredAuditLogs(_ context.Context, audit time.Time) (int64, error) {
	s.audit = audit
	return 2, nil
}
func TestLogRetentionBoundaries(t *testing.T) {
	now := time.Date(2026, 8, 6, 0, 0, 0, 0, time.UTC)
	p := &retentionPortStub{}
	u := newLogRetentionUsecase(p, LogRetentionConfig{SystemDays: 90, SensitiveDays: 180, AuditDays: 365}, func() time.Time { return now })
	r, err := u.Cleanup(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if r.SystemDeleted != 1 || r.AuditDeleted != 2 {
		t.Fatalf("result=%+v", r)
	}
	if !p.standard.Equal(now.AddDate(0, 0, -90)) || !p.sensitive.Equal(now.AddDate(0, 0, -180)) || !p.audit.Equal(now.AddDate(0, 0, -365)) {
		t.Fatalf("cutoffs: %+v", p)
	}
}

package entity

import (
	"testing"
	"time"
)

func TestRefreshTokenIsExpired(t *testing.T) {
	expiresAt := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{name: "before expiry", now: expiresAt.Add(-time.Nanosecond), want: false},
		{name: "at expiry boundary", now: expiresAt, want: true},
		{name: "after expiry", now: expiresAt.Add(time.Nanosecond), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &RefreshToken{ExpiresAt: expiresAt}
			if got := token.IsExpired(tt.now); got != tt.want {
				t.Fatalf("IsExpired(%s)=%v want=%v", tt.now, got, tt.want)
			}
		})
	}
}

func TestRefreshTokenIsUsed(t *testing.T) {
	now := time.Now().UTC()

	if (&RefreshToken{}).IsUsed() {
		t.Fatal("token without UsedAt must not be used")
	}
	if !(&RefreshToken{UsedAt: &now}).IsUsed() {
		t.Fatal("token with UsedAt must be used")
	}
}

func TestRefreshTokenIsRevoked(t *testing.T) {
	now := time.Now().UTC()

	if (&RefreshToken{}).IsRevoked() {
		t.Fatal("token without RevokedAt must not be revoked")
	}
	if !(&RefreshToken{RevokedAt: &now}).IsRevoked() {
		t.Fatal("token with RevokedAt must be revoked")
	}
}

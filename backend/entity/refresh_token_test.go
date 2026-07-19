package entity

import (
	"testing"
	"time"
)

func TestRefreshTokenState(t *testing.T) {
	now := time.Now().UTC()
	token := &RefreshToken{ExpiresAt: now.Add(time.Minute)}
	if !token.CanRotate(now) {
		t.Fatal("token should rotate")
	}
	used := now
	token.UsedAt = &used
	if token.CanRotate(now) {
		t.Fatal("used token must not rotate")
	}
	token.UsedAt = nil
	token.ExpiresAt = now
	if !token.IsExpired(now) {
		t.Fatal("same instant must be expired")
	}
}

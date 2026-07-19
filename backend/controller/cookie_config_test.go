package controller

import (
	"net/http"
	"testing"
)

func TestCookieConfigNoneRequiresSecure(t *testing.T) {
	t.Setenv("COOKIE_SAME_SITE", "none")
	t.Setenv("COOKIE_SECURE", "false")
	if _, err := LoadCookieConfigFromEnv(); err == nil {
		t.Fatal("expected error")
	}
	t.Setenv("COOKIE_SECURE", "true")
	cfg, err := LoadCookieConfigFromEnv()
	if err != nil || cfg.SameSite != http.SameSiteNoneMode || !cfg.Secure {
		t.Fatalf("cfg=%#v err=%v", cfg, err)
	}
}
func TestCookieConfigDomainAndNames(t *testing.T) {
	t.Setenv("COOKIE_DOMAIN", "example.com")
	t.Setenv("REFRESH_TOKEN_COOKIE_NAME", "custom_rt")
	t.Setenv("CSRF_COOKIE_NAME", "custom_csrf")
	cfg, err := LoadCookieConfigFromEnv()
	if err != nil || cfg.Domain != "example.com" || cfg.RefreshName != "custom_rt" || cfg.CSRFName != "custom_csrf" {
		t.Fatalf("cfg=%#v err=%v", cfg, err)
	}
}
func TestCookieConfigRejectsInvalidSameSite(t *testing.T) {
	t.Setenv("COOKIE_SAME_SITE", "invalid")
	if _, err := LoadCookieConfigFromEnv(); err == nil {
		t.Fatal("expected error")
	}
}

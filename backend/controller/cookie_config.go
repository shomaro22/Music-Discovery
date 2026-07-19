package controller

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type CookieConfig struct {
	RefreshName string
	CSRFName    string
	Domain      string
	Path        string
	Secure      bool
	SameSite    http.SameSite
}

func LoadCookieConfigFromEnv() (CookieConfig, error) {
	cfg := CookieConfig{
		RefreshName: envOrDefault("REFRESH_TOKEN_COOKIE_NAME", "refresh_token"),
		CSRFName:    envOrDefault("CSRF_COOKIE_NAME", "csrf_token"),
		Domain:      strings.TrimSpace(os.Getenv("COOKIE_DOMAIN")),
		Path:        "/api/auth",
	}

	appEnv := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	if appEnv == "" {
		appEnv = "development"
	}
	switch appEnv {
	case "development", "local", "test", "staging", "production":
	default:
		return CookieConfig{}, fmt.Errorf("invalid APP_ENV: %s", appEnv)
	}

	secureDefault := appEnv == "production" || appEnv == "staging"
	cfg.Secure = secureDefault
	if raw := strings.TrimSpace(os.Getenv("COOKIE_SECURE")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return CookieConfig{}, fmt.Errorf("invalid COOKIE_SECURE: %w", err)
		}
		cfg.Secure = value
	}

	switch strings.ToLower(strings.TrimSpace(os.Getenv("COOKIE_SAME_SITE"))) {
	case "", "lax":
		cfg.SameSite = http.SameSiteLaxMode
	case "strict":
		cfg.SameSite = http.SameSiteStrictMode
	case "none":
		cfg.SameSite = http.SameSiteNoneMode
		if !cfg.Secure {
			return CookieConfig{}, fmt.Errorf("COOKIE_SECURE must be true when COOKIE_SAME_SITE=none")
		}
	default:
		return CookieConfig{}, fmt.Errorf("invalid COOKIE_SAME_SITE")
	}

	if appEnv == "production" && !cfg.Secure {
		return CookieConfig{}, fmt.Errorf("COOKIE_SECURE must be true in production")
	}
	if cfg.RefreshName == "" || cfg.CSRFName == "" {
		return CookieConfig{}, fmt.Errorf("cookie names must not be empty")
	}
	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

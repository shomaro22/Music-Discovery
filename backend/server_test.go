package main

import (
	"strings"
	"testing"
)

func TestLoadRateLimitCapacityMultiplierAllowsDefaultOutsideTest(t *testing.T) {
	t.Setenv("RATE_LIMIT_CAPACITY_MULTIPLIER", "1")

	got, err := loadRateLimitCapacityMultiplier("production")
	if err != nil {
		t.Fatalf("loadRateLimitCapacityMultiplier returned error: %v", err)
	}
	if got != 1 {
		t.Fatalf("multiplier=%d want=1", got)
	}
}

func TestLoadRateLimitCapacityMultiplierAllowsOverrideOnlyInTest(t *testing.T) {
	t.Setenv("RATE_LIMIT_CAPACITY_MULTIPLIER", "1000")

	got, err := loadRateLimitCapacityMultiplier("test")
	if err != nil {
		t.Fatalf("loadRateLimitCapacityMultiplier returned error: %v", err)
	}
	if got != 1000 {
		t.Fatalf("multiplier=%d want=1000", got)
	}
}

func TestLoadRateLimitCapacityMultiplierRejectsOverrideOutsideTest(t *testing.T) {
	t.Setenv("RATE_LIMIT_CAPACITY_MULTIPLIER", "1000")

	_, err := loadRateLimitCapacityMultiplier("production")
	if err == nil {
		t.Fatal("expected production override to fail")
	}
	if !strings.Contains(err.Error(), "APP_ENV=test") {
		t.Fatalf("error=%q", err.Error())
	}
}

func TestLoadRateLimitCapacityMultiplierRejectsInvalidConfiguredValue(t *testing.T) {
	t.Setenv("RATE_LIMIT_CAPACITY_MULTIPLIER", "invalid")

	_, err := loadRateLimitCapacityMultiplier("production")
	if err == nil {
		t.Fatal("expected invalid multiplier to fail")
	}
}

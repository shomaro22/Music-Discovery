package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
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

type gracefulServerStub struct {
	started        chan struct{}
	shutdown       chan struct{}
	startErr       error
	shutdownErr    error
	shutdownCtxErr error
}

func (s *gracefulServerStub) Start(string) error {
	close(s.started)
	<-s.shutdown
	return s.startErr
}

func (s *gracefulServerStub) Shutdown(ctx context.Context) error {
	select {
	case <-ctx.Done():
		s.shutdownCtxErr = ctx.Err()
		return ctx.Err()
	default:
	}
	if s.shutdownErr != nil {
		return s.shutdownErr
	}
	close(s.shutdown)
	return nil
}

func TestServeWithGracefulShutdownStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	server := &gracefulServerStub{started: make(chan struct{}), shutdown: make(chan struct{}), startErr: http.ErrServerClosed}
	done := make(chan error, 1)
	go func() {
		done <- serveWithGracefulShutdown(ctx, server, ":0", time.Second)
	}()
	<-server.started
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestServeWithGracefulShutdownReturnsStartError(t *testing.T) {
	server := &immediateServerStub{startErr: errors.New("listen failed")}
	if err := serveWithGracefulShutdown(context.Background(), server, ":0", time.Second); err == nil || err.Error() != "listen failed" {
		t.Fatalf("err=%v", err)
	}
}

func TestServeWithGracefulShutdownReturnsShutdownError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	server := &gracefulServerStub{started: make(chan struct{}), shutdown: make(chan struct{}), shutdownErr: errors.New("shutdown failed")}
	done := make(chan error, 1)
	go func() {
		done <- serveWithGracefulShutdown(ctx, server, ":0", time.Second)
	}()
	<-server.started
	cancel()
	if err := <-done; err == nil || !strings.Contains(err.Error(), "shutdown failed") {
		t.Fatalf("err=%v", err)
	}
	close(server.shutdown)
}

type immediateServerStub struct{ startErr error }

func (s *immediateServerStub) Start(string) error             { return s.startErr }
func (s *immediateServerStub) Shutdown(context.Context) error { return nil }

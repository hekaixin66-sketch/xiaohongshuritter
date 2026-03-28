package main

import (
	"context"
	"testing"
	"time"
)

func TestBrowserGuardAcquireLimit(t *testing.T) {
	metrics := NewServiceMetrics()
	guard := NewBrowserGuard(1, metrics)

	release, err := guard.Acquire(context.Background())
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	defer release()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := guard.Acquire(ctx); err == nil {
		t.Fatal("expected second acquire to fail when browser slot is full")
	}

	release()
	if _, err := guard.Acquire(context.Background()); err != nil {
		t.Fatalf("expected acquire after release to succeed: %v", err)
	}
}

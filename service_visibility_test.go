package main

import (
	"context"
	"testing"
	"time"
)

func TestNormalizeVisibility(t *testing.T) {
	cases := []struct {
		in      string
		out     string
		wantErr bool
	}{
		{in: "", out: "公开可见"},
		{in: "public", out: "公开可见"},
		{in: "PUBLIC", out: "公开可见"},
		{in: "self-only", out: "仅自己可见"},
		{in: "private", out: "仅自己可见"},
		{in: "friends-only", out: "仅互关好友可见"},
		{in: "mutual_follow", out: "仅互关好友可见"},
		{in: "公开可见", out: "公开可见"},
		{in: "仅自己可见", out: "仅自己可见"},
		{in: "仅互关好友可见", out: "仅互关好友可见"},
		{in: "unknown", wantErr: true},
	}

	for _, tc := range cases {
		got, err := normalizeVisibility(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("normalizeVisibility(%q) expected error, got none", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("normalizeVisibility(%q) unexpected error: %v", tc.in, err)
		}
		if got != tc.out {
			t.Fatalf("normalizeVisibility(%q) = %q, want %q", tc.in, got, tc.out)
		}
	}
}

func TestLoginWatcherLifecycle(t *testing.T) {
	service := &XiaohongshuService{loginWatchers: make(map[string]*loginWatcher)}
	ctx, cancel := context.WithCancel(context.Background())
	watcher := &loginWatcher{
		img:       "img",
		timeout:   time.Minute,
		startedAt: time.Now(),
		cancel:    cancel,
	}

	service.setLoginWatcher("tenant/account", watcher)
	if got := service.getLoginWatcher("tenant/account"); got != watcher {
		t.Fatal("expected active watcher")
	}

	service.clearLoginWatcher("tenant/account")
	select {
	case <-ctx.Done():
	default:
		t.Fatal("expected watcher context to be canceled")
	}
	if got := service.getLoginWatcher("tenant/account"); got != nil {
		t.Fatal("expected watcher to be removed")
	}
}

func TestGetLoginWatcherRemovesExpiredWatcher(t *testing.T) {
	service := &XiaohongshuService{loginWatchers: make(map[string]*loginWatcher)}
	ctx, cancel := context.WithCancel(context.Background())
	watcher := &loginWatcher{
		img:       "img",
		timeout:   time.Minute,
		startedAt: time.Now().Add(-2 * time.Minute),
		cancel:    cancel,
	}

	service.setLoginWatcher("tenant/account", watcher)
	if got := service.getLoginWatcher("tenant/account"); got != nil {
		t.Fatal("expected expired watcher to be ignored")
	}
	select {
	case <-ctx.Done():
	default:
		t.Fatal("expected expired watcher context to be canceled")
	}
}

func TestGetOrCreateLoginWatcherReusesPendingWatcher(t *testing.T) {
	service := &XiaohongshuService{loginWatchers: make(map[string]*loginWatcher)}
	scope := AccountScope{TenantID: "tenant", AccountID: "account"}

	first, created := service.getOrCreateLoginWatcher(scope, time.Minute)
	if !created || first == nil {
		t.Fatal("expected first watcher to be created")
	}

	second, created := service.getOrCreateLoginWatcher(scope, time.Minute)
	if created {
		t.Fatal("expected existing watcher to be reused")
	}
	if second != first {
		t.Fatal("expected same watcher instance")
	}
}

func TestGetOrCreateLoginWatcherReplacesExpiredWatcher(t *testing.T) {
	service := &XiaohongshuService{loginWatchers: make(map[string]*loginWatcher)}
	scope := AccountScope{TenantID: "tenant", AccountID: "account"}
	ctx, cancel := context.WithCancel(context.Background())
	stale := &loginWatcher{
		timeout:   time.Minute,
		startedAt: time.Now().Add(-2 * time.Minute),
		cancel:    cancel,
		ready:     make(chan struct{}),
	}
	service.loginWatchers[scope.Label()] = stale

	fresh, created := service.getOrCreateLoginWatcher(scope, time.Minute)
	if !created || fresh == nil {
		t.Fatal("expected fresh watcher to be created")
	}
	if fresh == stale {
		t.Fatal("expected stale watcher to be replaced")
	}
	select {
	case <-ctx.Done():
	default:
		t.Fatal("expected stale watcher to be canceled")
	}
}

func TestLoginWatcherWaitUntilReadyHonorsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	watcher := &loginWatcher{ready: make(chan struct{})}
	if err := watcher.waitUntilReady(ctx); err == nil {
		t.Fatal("expected context error")
	}
}

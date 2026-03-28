package main

import (
	"context"
	"testing"
	"time"
)

func TestAccountManagerResolveDefault(t *testing.T) {
	cfg := EnterpriseAccountConfig{
		DefaultTenant:        "tenant-a",
		DefaultAccount:       "account-a",
		GlobalMaxConcurrency: 4,
		Tenants: []EnterpriseTenantConfig{
			{
				ID:             "tenant-a",
				DefaultAccount: "account-a",
				Accounts: []EnterpriseProfileConfig{
					{ID: "account-a", CookiePath: "./data/a.json", MaxConcurrency: 2},
					{ID: "account-b", CookiePath: "./data/b.json", MaxConcurrency: 2},
				},
			},
		},
	}

	manager, err := buildAccountManager("test.json", cfg, 2)
	if err != nil {
		t.Fatalf("build manager failed: %v", err)
	}

	resolved, err := manager.Resolve(AccountScope{})
	if err != nil {
		t.Fatalf("resolve default failed: %v", err)
	}
	if resolved.TenantID != "tenant-a" || resolved.AccountID != "account-a" {
		t.Fatalf("unexpected default resolve: %+v", resolved)
	}
}

func TestAccountManagerAcquireLimit(t *testing.T) {
	cfg := EnterpriseAccountConfig{
		DefaultTenant:        "tenant-a",
		DefaultAccount:       "account-a",
		GlobalMaxConcurrency: 2,
		Tenants: []EnterpriseTenantConfig{
			{
				ID:             "tenant-a",
				DefaultAccount: "account-a",
				Accounts: []EnterpriseProfileConfig{
					{ID: "account-a", CookiePath: "./data/a.json", MaxConcurrency: 1},
				},
			},
		},
	}

	manager, err := buildAccountManager("test.json", cfg, 1)
	if err != nil {
		t.Fatalf("build manager failed: %v", err)
	}
	manager.acquireTimeout = 100 * time.Millisecond

	first, err := manager.Acquire(context.Background(), AccountScope{})
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	defer first.Release()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := manager.Acquire(ctx, AccountScope{}); err == nil {
		t.Fatal("second acquire should fail when account max concurrency is reached")
	}

	first.Release()
	if _, err := manager.Acquire(context.Background(), AccountScope{}); err != nil {
		t.Fatalf("acquire after release failed: %v", err)
	}
}

func TestAccountManagerCooldownAfterRepeatedRetryableFailures(t *testing.T) {
	cfg := EnterpriseAccountConfig{
		DefaultTenant:        "tenant-a",
		DefaultAccount:       "account-a",
		GlobalMaxConcurrency: 2,
		Tenants: []EnterpriseTenantConfig{
			{
				ID:             "tenant-a",
				DefaultAccount: "account-a",
				Accounts: []EnterpriseProfileConfig{
					{ID: "account-a", CookiePath: "./data/a.json", MaxConcurrency: 1},
				},
			},
		},
	}

	manager, err := buildAccountManager("test.json", cfg, 1)
	if err != nil {
		t.Fatalf("build manager failed: %v", err)
	}
	manager.cooldownAfter = 2
	manager.cooldownPeriod = time.Minute

	scope := AccountScope{TenantID: "tenant-a", AccountID: "account-a"}
	retryableErr := newAppError("OPERATION_TIMEOUT", "operation timed out", 504, true, context.DeadlineExceeded, nil)

	manager.RecordPublishResult(scope, retryableErr)
	manager.RecordPublishResult(scope, retryableErr)

	_, err = manager.Acquire(context.Background(), scope)
	if err == nil {
		t.Fatal("expected acquire to fail during cooldown")
	}

	appErr, ok := asAppError(err)
	if !ok {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != "ACCOUNT_COOLDOWN" {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}

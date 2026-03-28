package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecommendAccountsForPublishPrefersReadyAccount(t *testing.T) {
	tempDir := t.TempDir()
	readyCookie := filepath.Join(tempDir, "ready.json")
	if err := os.WriteFile(readyCookie, []byte(`[]`), 0644); err != nil {
		t.Fatalf("write cookie failed: %v", err)
	}

	cfg := EnterpriseAccountConfig{
		DefaultTenant:        "tenant-a",
		DefaultAccount:       "ready",
		GlobalMaxConcurrency: 4,
		Tenants: []EnterpriseTenantConfig{
			{
				ID:             "tenant-a",
				DefaultAccount: "ready",
				Accounts: []EnterpriseProfileConfig{
					{ID: "ready", CookiePath: readyCookie, MaxConcurrency: 1},
					{ID: "cold", CookiePath: filepath.Join(tempDir, "cold.json"), MaxConcurrency: 1},
				},
			},
		},
	}

	manager, err := buildAccountManager("test.json", cfg, 1)
	if err != nil {
		t.Fatalf("build manager failed: %v", err)
	}
	manager.cooldownAfter = 1
	manager.cooldownPeriod = 10 * time.Minute
	manager.RecordPublishResult(AccountScope{TenantID: "tenant-a", AccountID: "cold"}, newAppError("OPERATION_TIMEOUT", "operation timed out", 504, true, nil, nil))

	service := &XiaohongshuService{
		accountManager: manager,
		metrics:        NewServiceMetrics(),
	}
	appServer := NewAppServer(service)

	result := appServer.RecommendAccountsForPublish(SchedulerRecommendationRequest{
		AccountScope: AccountScope{TenantID: "tenant-a"},
		Limit:        2,
	})

	if result.SelectedAccount == nil {
		t.Fatal("expected selected account")
	}
	if result.SelectedAccount.AccountID != "ready" {
		t.Fatalf("expected ready account to be selected, got %s", result.SelectedAccount.AccountID)
	}
	if !result.SelectedAccount.CookiePresent {
		t.Fatal("expected selected account to have cookie")
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(result.Candidates))
	}
	if result.Candidates[1].Available {
		t.Fatal("expected cold account to be unavailable due to cooldown/cookie state")
	}
}

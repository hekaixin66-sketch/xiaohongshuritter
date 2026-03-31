package main

import (
	"context"
	"testing"
	"time"
)

func TestPublishJobManagerSubmitContentLifecycle(t *testing.T) {
	manager := NewPublishJobManager(func(ctx context.Context, req *PublishRequest) (*PublishExecutionResult, error) {
		return &PublishExecutionResult{
			OK:        true,
			Status:    "succeeded",
			TaskID:    req.TaskID,
			TenantID:  AccountScopeFromContext(ctx).TenantID,
			AccountID: AccountScopeFromContext(ctx).AccountID,
			Title:     req.Title,
		}, nil
	}, nil)

	resp, err := manager.SubmitContent(AccountScope{TenantID: "default", AccountID: "main"}, &PublishRequest{
		Title:   "hello",
		Content: "world",
		Images:  []string{"a.jpg"},
	})
	if err != nil {
		t.Fatalf("SubmitContent failed: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status, err := manager.Get(resp.JobID)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if status.Status == "succeeded" {
			if status.Result == nil || status.Result.JobID != resp.JobID {
				t.Fatal("expected final result with job id")
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatal("job did not reach succeeded state in time")
}

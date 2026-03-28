package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPublishJobManagerSubmitContentLifecycle(t *testing.T) {
	manager := NewPublishJobManager(nil, func(ctx context.Context, req *PublishRequest) (*PublishExecutionResult, error) {
		return &PublishExecutionResult{
			OK:         true,
			Mode:       string(PublishModeSync),
			Operation:  string(OperationPublishContent),
			TaskID:     req.TaskID,
			BatchID:    req.BatchID,
			TenantID:   AccountScopeFromContext(ctx).TenantID,
			AccountID:  AccountScopeFromContext(ctx).AccountID,
			Title:      req.Title,
			Status:     "succeeded",
			DurationMs: 10,
		}, nil
	}, nil, func(scope AccountScope) (ResolvedAccount, error) {
		return ResolvedAccount{
			TenantID:       scope.TenantID,
			AccountID:      scope.AccountID,
			MaxConcurrency: 1,
		}, nil
	})

	scope := AccountScope{TenantID: "tenant-a", AccountID: "account-a"}
	accepted, err := manager.SubmitContent(scope, &PublishRequest{
		Title:   "hello",
		Content: "world",
		Images:  []string{"img-a"},
		BatchID: "batch-1",
	})
	if err != nil {
		t.Fatalf("SubmitContent failed: %v", err)
	}
	if !accepted.Accepted {
		t.Fatal("expected accepted response")
	}

	result := waitForJobStatus(t, manager, accepted.JobID, PublishJobStatusSucceeded)
	if result.Result == nil {
		t.Fatal("expected async result payload")
	}
	if result.Result.Mode != string(PublishModeAsync) {
		t.Fatalf("expected async mode, got %s", result.Result.Mode)
	}
	if result.Result.TaskID == "" {
		t.Fatal("expected propagated task id")
	}
}

func TestPublishJobManagerSubmitVideoFailure(t *testing.T) {
	manager := NewPublishJobManager(nil, nil, func(ctx context.Context, req *PublishVideoRequest) (*PublishExecutionResult, error) {
		result := &PublishExecutionResult{
			OK:         false,
			Mode:       string(PublishModeSync),
			Operation:  string(OperationPublishVideo),
			TaskID:     req.TaskID,
			BatchID:    req.BatchID,
			TenantID:   AccountScopeFromContext(ctx).TenantID,
			AccountID:  AccountScopeFromContext(ctx).AccountID,
			Title:      req.Title,
			Status:     "failed",
			ErrorCode:  "OPERATION_TIMEOUT",
			Retryable:  true,
			DurationMs: 10,
		}
		return result, context.DeadlineExceeded
	}, func(scope AccountScope) (ResolvedAccount, error) {
		return ResolvedAccount{
			TenantID:       scope.TenantID,
			AccountID:      scope.AccountID,
			MaxConcurrency: 1,
		}, nil
	})

	scope := AccountScope{TenantID: "tenant-b", AccountID: "account-b"}
	accepted, err := manager.SubmitVideo(scope, &PublishVideoRequest{
		Title:   "video",
		Content: "body",
		Video:   "demo.mp4",
	})
	if err != nil {
		t.Fatalf("SubmitVideo failed: %v", err)
	}

	result := waitForJobStatus(t, manager, accepted.JobID, PublishJobStatusFailed)
	if result.Result == nil {
		t.Fatal("expected async result payload")
	}
	if !result.Result.Retryable {
		t.Fatal("expected retryable failure")
	}
}

func TestPublishJobManagerGetMissingJob(t *testing.T) {
	manager := NewPublishJobManager(nil, nil, nil, func(scope AccountScope) (ResolvedAccount, error) {
		return ResolvedAccount{TenantID: scope.TenantID, AccountID: scope.AccountID, MaxConcurrency: 1}, nil
	})

	_, err := manager.Get("missing")
	if err == nil {
		t.Fatal("expected missing job error")
	}

	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected AppError, got %T", err)
	}
	if appErr.Code != "JOB_NOT_FOUND" {
		t.Fatalf("unexpected error code: %s", appErr.Code)
	}
}

func TestPublishJobManagerQueueFullRejects(t *testing.T) {
	t.Setenv("XHS_ACCOUNT_QUEUE_CAPACITY", "1")
	t.Setenv("XHS_MAX_OUTSTANDING_JOBS", "2")

	blocker := make(chan struct{})
	started := make(chan struct{})
	manager := NewPublishJobManager(nil, func(ctx context.Context, req *PublishRequest) (*PublishExecutionResult, error) {
		select {
		case <-started:
		default:
			close(started)
		}
		<-blocker
		return &PublishExecutionResult{
			OK:        true,
			Mode:      string(PublishModeSync),
			Operation: string(OperationPublishContent),
			TaskID:    req.TaskID,
			Status:    "succeeded",
		}, nil
	}, nil, func(scope AccountScope) (ResolvedAccount, error) {
		return ResolvedAccount{
			TenantID:       scope.TenantID,
			AccountID:      scope.AccountID,
			MaxConcurrency: 1,
		}, nil
	})
	defer close(blocker)

	scope := AccountScope{TenantID: "tenant-a", AccountID: "account-a"}
	if _, err := manager.SubmitContent(scope, &PublishRequest{Title: "a", Content: "b", Images: []string{"1"}}); err != nil {
		t.Fatalf("first submit failed: %v", err)
	}
	<-started
	if _, err := manager.SubmitContent(scope, &PublishRequest{Title: "c", Content: "d", Images: []string{"2"}}); err != nil {
		t.Fatalf("second submit failed: %v", err)
	}
	if _, err := manager.SubmitContent(scope, &PublishRequest{Title: "e", Content: "f", Images: []string{"3"}}); err == nil {
		t.Fatal("expected queue full error on third submit")
	}
}

func waitForJobStatus(t *testing.T, manager *PublishJobManager, jobID string, want PublishJobStatus) *PublishJobStatusResponse {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		result, err := manager.Get(jobID)
		if err != nil {
			t.Fatalf("Get job status failed: %v", err)
		}
		if result.Status == want {
			return result
		}
		time.Sleep(10 * time.Millisecond)
	}

	result, err := manager.Get(jobID)
	if err != nil {
		t.Fatalf("Get job status failed: %v", err)
	}
	t.Fatalf("job did not reach status %s, current=%s", want, result.Status)
	return nil
}

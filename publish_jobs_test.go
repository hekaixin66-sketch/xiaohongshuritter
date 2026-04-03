package main

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestPublishJobManagerSubmitContentLifecycle(t *testing.T) {
	manager := newPublishJobManager(func(ctx context.Context, req *PublishRequest) (*PublishExecutionResult, error) {
		return &PublishExecutionResult{
			OK:        true,
			Status:    publishJobStatusSucceeded,
			TaskID:    req.TaskID,
			TenantID:  AccountScopeFromContext(ctx).TenantID,
			AccountID: AccountScopeFromContext(ctx).AccountID,
			Title:     req.Title,
		}, nil
	}, nil, publishJobManagerConfig{
		workerCount:     1,
		queueSize:       1,
		cleanupInterval: time.Hour,
	})
	t.Cleanup(manager.Close)

	resp, err := manager.SubmitContent(AccountScope{TenantID: "default", AccountID: "main"}, &PublishRequest{
		Title:   "hello",
		Content: "world",
		Images:  []string{"a.jpg"},
	})
	if err != nil {
		t.Fatalf("SubmitContent failed: %v", err)
	}

	status := waitForPublishJobStatus(t, manager, resp.JobID, publishJobStatusSucceeded)
	if status.Result == nil || status.Result.JobID != resp.JobID {
		t.Fatal("expected final result with job id")
	}
	if status.Result.Mode != string(PublishModeAsync) {
		t.Fatalf("unexpected result mode: %s", status.Result.Mode)
	}
}

func TestPublishJobManagerSubmitContentQueueFull(t *testing.T) {
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var releaseOnce sync.Once
	manager := newPublishJobManager(func(ctx context.Context, req *PublishRequest) (*PublishExecutionResult, error) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		return &PublishExecutionResult{
			OK:     true,
			Status: publishJobStatusSucceeded,
			TaskID: req.TaskID,
			Title:  req.Title,
		}, nil
	}, nil, publishJobManagerConfig{
		workerCount:     1,
		queueSize:       1,
		cleanupInterval: time.Hour,
	})
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
		manager.Close()
	})

	scope := AccountScope{TenantID: "default", AccountID: "main"}
	first, err := manager.SubmitContent(scope, &PublishRequest{Title: "first"})
	if err != nil {
		t.Fatalf("SubmitContent first failed: %v", err)
	}
	<-started

	second, err := manager.SubmitContent(scope, &PublishRequest{Title: "second"})
	if err != nil {
		t.Fatalf("SubmitContent second failed: %v", err)
	}
	if second == nil || second.JobID == "" {
		t.Fatal("expected second queued job")
	}

	_, err = manager.SubmitContent(scope, &PublishRequest{Title: "third"})
	if !errors.Is(err, ErrPublishJobQueueFull) {
		t.Fatalf("expected queue full error, got %v", err)
	}

	manager.mu.RLock()
	jobCount := len(manager.jobs)
	manager.mu.RUnlock()
	if jobCount != 2 {
		t.Fatalf("expected 2 retained jobs, got %d", jobCount)
	}

	releaseOnce.Do(func() { close(release) })
	waitForPublishJobStatus(t, manager, first.JobID, publishJobStatusSucceeded)
	waitForPublishJobStatus(t, manager, second.JobID, publishJobStatusSucceeded)
}

func TestPublishJobManagerCloseAllowsFastRunningJobToFinish(t *testing.T) {
	started := make(chan struct{}, 1)
	manager := newPublishJobManager(func(ctx context.Context, req *PublishRequest) (*PublishExecutionResult, error) {
		select {
		case started <- struct{}{}:
		default:
		}
		time.Sleep(20 * time.Millisecond)
		return &PublishExecutionResult{
			OK:     true,
			Status: publishJobStatusSucceeded,
			TaskID: req.TaskID,
			Title:  req.Title,
		}, nil
	}, nil, publishJobManagerConfig{
		workerCount:     1,
		queueSize:       1,
		cleanupInterval: time.Hour,
		closeTimeout:    time.Second,
	})

	resp, err := manager.SubmitContent(AccountScope{TenantID: "default", AccountID: "main"}, &PublishRequest{Title: "fast"})
	if err != nil {
		t.Fatalf("SubmitContent failed: %v", err)
	}
	<-started

	manager.Close()

	status := waitForPublishJobStatus(t, manager, resp.JobID, publishJobStatusSucceeded)
	if status.Result == nil || !status.Result.OK {
		t.Fatal("expected fast running job to finish successfully during close")
	}
}

func TestPublishJobManagerCloseFailsPendingJobs(t *testing.T) {
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var releaseOnce sync.Once
	manager := newPublishJobManager(func(ctx context.Context, req *PublishRequest) (*PublishExecutionResult, error) {
		select {
		case started <- struct{}{}:
		default:
		}
		<-release
		return &PublishExecutionResult{
			OK:     true,
			Status: publishJobStatusSucceeded,
			TaskID: req.TaskID,
			Title:  req.Title,
		}, nil
	}, nil, publishJobManagerConfig{
		workerCount:     1,
		queueSize:       2,
		cleanupInterval: time.Hour,
		closeTimeout:    50 * time.Millisecond,
	})
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
		manager.Close()
	})

	scope := AccountScope{TenantID: "default", AccountID: "main"}
	first, err := manager.SubmitContent(scope, &PublishRequest{Title: "first"})
	if err != nil {
		t.Fatalf("SubmitContent first failed: %v", err)
	}
	<-started

	second, err := manager.SubmitContent(scope, &PublishRequest{Title: "second"})
	if err != nil {
		t.Fatalf("SubmitContent second failed: %v", err)
	}

	closed := make(chan struct{})
	go func() {
		manager.Close()
		close(closed)
	}()

	select {
	case <-closed:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("expected Close to return within bounded timeout")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, err = manager.SubmitContent(scope, &PublishRequest{Title: "third"})
		if errors.Is(err, ErrPublishJobManagerClosed) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !errors.Is(err, ErrPublishJobManagerClosed) {
		t.Fatalf("expected manager closed error, got %v", err)
	}

	firstStatus := waitForPublishJobStatus(t, manager, first.JobID, publishJobStatusFailed)
	if firstStatus.Result == nil {
		t.Fatal("expected failed running result")
	}
	if firstStatus.Result.ErrorCode != "PUBLISH_JOB_SHUTDOWN_TIMEOUT" {
		t.Fatalf("unexpected running job error code: %s", firstStatus.Result.ErrorCode)
	}

	secondStatus := waitForPublishJobStatus(t, manager, second.JobID, publishJobStatusFailed)
	if secondStatus.Result == nil {
		t.Fatal("expected failed pending result")
	}
	if secondStatus.Result.ErrorCode != "PUBLISH_JOB_SHUTDOWN_TIMEOUT" {
		t.Fatalf("unexpected pending job error code: %s", secondStatus.Result.ErrorCode)
	}
	if secondStatus.Result.ErrorMessage != ErrPublishJobShutdownTimeout.Error() {
		t.Fatalf("unexpected pending job error message: %s", secondStatus.Result.ErrorMessage)
	}

	releaseOnce.Do(func() { close(release) })
}

func TestPublishJobManagerCleanupExpiredJobs(t *testing.T) {
	current := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)
	var nowMu sync.Mutex

	manager := newPublishJobManager(func(ctx context.Context, req *PublishRequest) (*PublishExecutionResult, error) {
		return &PublishExecutionResult{
			OK:     true,
			Status: publishJobStatusSucceeded,
			TaskID: req.TaskID,
		}, nil
	}, nil, publishJobManagerConfig{
		workerCount:     1,
		queueSize:       1,
		jobTTL:          time.Minute,
		cleanupInterval: time.Hour,
	})
	t.Cleanup(manager.Close)
	manager.now = func() time.Time {
		nowMu.Lock()
		defer nowMu.Unlock()
		return current
	}

	resp, err := manager.SubmitContent(AccountScope{TenantID: "default", AccountID: "main"}, &PublishRequest{Title: "hello"})
	if err != nil {
		t.Fatalf("SubmitContent failed: %v", err)
	}
	waitForPublishJobStatus(t, manager, resp.JobID, publishJobStatusSucceeded)

	nowMu.Lock()
	current = current.Add(2 * time.Minute)
	nowMu.Unlock()

	if removed := manager.cleanupExpiredJobs(); removed != 1 {
		t.Fatalf("expected 1 cleaned job, got %d", removed)
	}
	if _, err := manager.Get(resp.JobID); err == nil {
		t.Fatal("expected cleaned job to be removed")
	}
}

func waitForPublishJobStatus(t *testing.T, manager *PublishJobManager, jobID, expected string) *PublishJobStatusResponse {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		status, err := manager.Get(jobID)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if status.Status == expected {
			return status
		}
		time.Sleep(20 * time.Millisecond)
	}

	t.Fatalf("job %s did not reach %s state in time", jobID, expected)
	return nil
}

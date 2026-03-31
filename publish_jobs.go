package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type publishContentExecutor func(context.Context, *PublishRequest) (*PublishExecutionResult, error)
type publishVideoExecutor func(context.Context, *PublishVideoRequest) (*PublishExecutionResult, error)

type publishJob struct {
	status PublishJobStatusResponse
}

type PublishJobManager struct {
	mu      sync.RWMutex
	jobs    map[string]*publishJob
	now     func() time.Time
	content publishContentExecutor
	video   publishVideoExecutor
}

func NewPublishJobManager(content publishContentExecutor, video publishVideoExecutor) *PublishJobManager {
	return &PublishJobManager{
		jobs:    make(map[string]*publishJob),
		now:     time.Now,
		content: content,
		video:   video,
	}
}

func (m *PublishJobManager) SubmitContent(scope AccountScope, req *PublishRequest) (*PublishAcceptedResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("publish request is required")
	}
	if m.content == nil {
		return nil, fmt.Errorf("content publish executor unavailable")
	}

	jobID := m.newJobID()
	taskID := ensureTaskID(req.TaskID, "pub")
	createdAt := formatRFC3339(m.now())

	reqCopy := *req
	reqCopy.TaskID = taskID
	reqCopy.Mode = string(PublishModeAsync)

	m.mu.Lock()
	m.jobs[jobID] = &publishJob{
		status: PublishJobStatusResponse{
			JobID:     jobID,
			Status:    "accepted",
			TaskID:    taskID,
			BatchID:   req.BatchID,
			TenantID:  scope.TenantID,
			AccountID: scope.AccountID,
			Mode:      string(PublishModeAsync),
			CreatedAt: createdAt,
		},
	}
	m.mu.Unlock()

	go m.runContent(scope, jobID, &reqCopy)

	return &PublishAcceptedResponse{
		Accepted:  true,
		JobID:     jobID,
		Status:    "accepted",
		TaskID:    taskID,
		BatchID:   req.BatchID,
		TenantID:  scope.TenantID,
		AccountID: scope.AccountID,
		Mode:      string(PublishModeAsync),
		CreatedAt: createdAt,
	}, nil
}

func (m *PublishJobManager) SubmitVideo(scope AccountScope, req *PublishVideoRequest) (*PublishAcceptedResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("publish request is required")
	}
	if m.video == nil {
		return nil, fmt.Errorf("video publish executor unavailable")
	}

	jobID := m.newJobID()
	taskID := ensureTaskID(req.TaskID, "pub")
	createdAt := formatRFC3339(m.now())

	reqCopy := *req
	reqCopy.TaskID = taskID
	reqCopy.Mode = string(PublishModeAsync)

	m.mu.Lock()
	m.jobs[jobID] = &publishJob{
		status: PublishJobStatusResponse{
			JobID:     jobID,
			Status:    "accepted",
			TaskID:    taskID,
			BatchID:   req.BatchID,
			TenantID:  scope.TenantID,
			AccountID: scope.AccountID,
			Mode:      string(PublishModeAsync),
			CreatedAt: createdAt,
		},
	}
	m.mu.Unlock()

	go m.runVideo(scope, jobID, &reqCopy)

	return &PublishAcceptedResponse{
		Accepted:  true,
		JobID:     jobID,
		Status:    "accepted",
		TaskID:    taskID,
		BatchID:   req.BatchID,
		TenantID:  scope.TenantID,
		AccountID: scope.AccountID,
		Mode:      string(PublishModeAsync),
		CreatedAt: createdAt,
	}, nil
}

func (m *PublishJobManager) Get(jobID string) (*PublishJobStatusResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, ok := m.jobs[strings.TrimSpace(jobID)]
	if !ok {
		return nil, fmt.Errorf("publish job %q not found", jobID)
	}

	copyStatus := job.status
	if job.status.Result != nil {
		copyResult := *job.status.Result
		copyStatus.Result = &copyResult
	}

	return &copyStatus, nil
}

func (m *PublishJobManager) runContent(scope AccountScope, jobID string, req *PublishRequest) {
	started := m.now()
	ctx, cancel := context.WithTimeout(WithAccountScope(context.Background(), scope), 10*time.Minute)
	defer cancel()

	m.markRunning(jobID, started)
	result, err := m.content(ctx, req)
	m.markFinished(jobID, started, result, err)
}

func (m *PublishJobManager) runVideo(scope AccountScope, jobID string, req *PublishVideoRequest) {
	started := m.now()
	ctx, cancel := context.WithTimeout(WithAccountScope(context.Background(), scope), 10*time.Minute)
	defer cancel()

	m.markRunning(jobID, started)
	result, err := m.video(ctx, req)
	m.markFinished(jobID, started, result, err)
}

func (m *PublishJobManager) markRunning(jobID string, started time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return
	}
	job.status.Status = "running"
	job.status.StartedAt = formatRFC3339(started)
}

func (m *PublishJobManager) markFinished(jobID string, started time.Time, result *PublishExecutionResult, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return
	}

	finished := m.now()
	job.status.StartedAt = firstNonEmpty(job.status.StartedAt, formatRFC3339(started))
	job.status.FinishedAt = formatRFC3339(finished)

	if result != nil {
		resultCopy := *result
		resultCopy.JobID = jobID
		resultCopy.Mode = string(PublishModeAsync)
		job.status.Result = &resultCopy
	}

	if err != nil {
		job.status.Status = "failed"
		if job.status.Result != nil {
			if job.status.Result.ErrorCode == "" {
				job.status.Result.ErrorCode = "PUBLISH_FAILED"
			}
			if job.status.Result.ErrorMessage == "" {
				job.status.Result.ErrorMessage = err.Error()
			}
			job.status.Result.OK = false
		}
		return
	}

	job.status.Status = "succeeded"
	if job.status.Result != nil {
		job.status.Result.JobID = jobID
		job.status.Result.Mode = string(PublishModeAsync)
	}
}

func (m *PublishJobManager) newJobID() string {
	return fmt.Sprintf("xhs_%d", m.now().UnixNano())
}

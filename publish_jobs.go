package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type operationTimeoutFunc func(context.Context, OperationName) (context.Context, context.CancelFunc)
type publishContentExecutor func(context.Context, *PublishRequest) (*PublishExecutionResult, error)
type publishVideoExecutor func(context.Context, *PublishVideoRequest) (*PublishExecutionResult, error)
type accountResolver func(AccountScope) (ResolvedAccount, error)

type publishJobRecord struct {
	response PublishJobStatusResponse
}

type publishJobTask struct {
	jobID string
	run   func()
}

type accountJobQueue struct {
	scope    AccountScope
	key      string
	tasks    chan *publishJobTask
	workers  int
	capacity int

	mu         sync.RWMutex
	activeJobs int
}

type AccountJobQueueStats struct {
	TenantID      string `json:"tenant_id"`
	AccountID     string `json:"account_id"`
	QueueDepth    int    `json:"queue_depth"`
	QueueCapacity int    `json:"queue_capacity"`
	ActiveJobs    int    `json:"active_jobs"`
	Workers       int    `json:"workers"`
}

type PublishJobRuntimeStats struct {
	OutstandingJobs int `json:"outstanding_jobs"`
	QueuedJobs      int `json:"queued_jobs"`
	ActiveJobs      int `json:"active_jobs"`
	QueueCount      int `json:"queue_count"`
	MaxOutstanding  int `json:"max_outstanding_jobs"`
}

type PublishJobManager struct {
	mu sync.RWMutex

	jobs        map[string]*publishJobRecord
	queues      map[string]*accountJobQueue
	ttl         time.Duration
	now         func() time.Time
	withTimeout operationTimeoutFunc
	runContent  publishContentExecutor
	runVideo    publishVideoExecutor

	resolveAccount       accountResolver
	accountQueueCapacity int
	maxOutstandingJobs   int
	outstandingJobs      int
}

func NewPublishJobManager(withTimeout operationTimeoutFunc, runContent publishContentExecutor, runVideo publishVideoExecutor, resolveAccount accountResolver) *PublishJobManager {
	ttl := parsePositiveDurationEnv("XHS_PUBLISH_JOB_TTL", 24*time.Hour)
	return &PublishJobManager{
		jobs:                 make(map[string]*publishJobRecord),
		queues:               make(map[string]*accountJobQueue),
		ttl:                  ttl,
		now:                  time.Now,
		withTimeout:          withTimeout,
		runContent:           runContent,
		runVideo:             runVideo,
		resolveAccount:       resolveAccount,
		accountQueueCapacity: parsePositiveIntEnv("XHS_ACCOUNT_QUEUE_CAPACITY", 20),
		maxOutstandingJobs:   parsePositiveIntEnv("XHS_MAX_OUTSTANDING_JOBS", 200),
	}
}

func (m *PublishJobManager) SubmitContent(scope AccountScope, req *PublishRequest) (*PublishAcceptedResponse, error) {
	if req == nil {
		return nil, newAppError("INVALID_REQUEST", "publish request is required", 400, false, nil, nil)
	}
	if m.runContent == nil {
		return nil, newAppError("ASYNC_NOT_AVAILABLE", "async content publish executor unavailable", 500, false, nil, nil)
	}

	queue, err := m.ensureQueue(scope)
	if err != nil {
		return nil, err
	}

	reqCopy := clonePublishRequest(req)
	taskID := ensureTaskID(reqCopy.TaskID, "pub")
	reqCopy.TaskID = taskID
	reqCopy.Mode = string(PublishModeAsync)

	jobID := m.newJobID()
	now := m.now().UTC()
	record := &publishJobRecord{
		response: PublishJobStatusResponse{
			Accepted:  true,
			JobID:     jobID,
			Status:    PublishJobStatusAccepted,
			Mode:      string(PublishModeAsync),
			Operation: string(OperationPublishContent),
			TaskID:    taskID,
			BatchID:   reqCopy.BatchID,
			TenantID:  scope.TenantID,
			AccountID: scope.AccountID,
			CreatedAt: formatTime(now),
		},
	}
	task := &publishJobTask{
		jobID: jobID,
		run: func() {
			m.runContentJob(jobID, scope, reqCopy)
		},
	}

	if err := m.enqueueJob(jobID, record, queue, task); err != nil {
		return nil, err
	}

	stats := queue.snapshot()
	return &PublishAcceptedResponse{
		Accepted:      true,
		JobID:         jobID,
		Status:        PublishJobStatusAccepted,
		Mode:          string(PublishModeAsync),
		Operation:     string(OperationPublishContent),
		TaskID:        taskID,
		BatchID:       reqCopy.BatchID,
		TenantID:      scope.TenantID,
		AccountID:     scope.AccountID,
		CreatedAt:     formatTime(now),
		QueueDepth:    stats.QueueDepth,
		QueueCapacity: stats.QueueCapacity,
		ActiveJobs:    stats.ActiveJobs,
	}, nil
}

func (m *PublishJobManager) SubmitVideo(scope AccountScope, req *PublishVideoRequest) (*PublishAcceptedResponse, error) {
	if req == nil {
		return nil, newAppError("INVALID_REQUEST", "publish request is required", 400, false, nil, nil)
	}
	if m.runVideo == nil {
		return nil, newAppError("ASYNC_NOT_AVAILABLE", "async video publish executor unavailable", 500, false, nil, nil)
	}

	queue, err := m.ensureQueue(scope)
	if err != nil {
		return nil, err
	}

	reqCopy := clonePublishVideoRequest(req)
	taskID := ensureTaskID(reqCopy.TaskID, "pub")
	reqCopy.TaskID = taskID
	reqCopy.Mode = string(PublishModeAsync)

	jobID := m.newJobID()
	now := m.now().UTC()
	record := &publishJobRecord{
		response: PublishJobStatusResponse{
			Accepted:  true,
			JobID:     jobID,
			Status:    PublishJobStatusAccepted,
			Mode:      string(PublishModeAsync),
			Operation: string(OperationPublishVideo),
			TaskID:    taskID,
			BatchID:   reqCopy.BatchID,
			TenantID:  scope.TenantID,
			AccountID: scope.AccountID,
			CreatedAt: formatTime(now),
		},
	}
	task := &publishJobTask{
		jobID: jobID,
		run: func() {
			m.runVideoJob(jobID, scope, reqCopy)
		},
	}

	if err := m.enqueueJob(jobID, record, queue, task); err != nil {
		return nil, err
	}

	stats := queue.snapshot()
	return &PublishAcceptedResponse{
		Accepted:      true,
		JobID:         jobID,
		Status:        PublishJobStatusAccepted,
		Mode:          string(PublishModeAsync),
		Operation:     string(OperationPublishVideo),
		TaskID:        taskID,
		BatchID:       reqCopy.BatchID,
		TenantID:      scope.TenantID,
		AccountID:     scope.AccountID,
		CreatedAt:     formatTime(now),
		QueueDepth:    stats.QueueDepth,
		QueueCapacity: stats.QueueCapacity,
		ActiveJobs:    stats.ActiveJobs,
	}, nil
}

func (m *PublishJobManager) Get(jobID string) (*PublishJobStatusResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.pruneExpiredLocked()
	record, ok := m.jobs[jobID]
	if !ok {
		return nil, newAppError("JOB_NOT_FOUND", "publish job not found", 404, false, nil, map[string]any{"job_id": jobID})
	}
	resp := clonePublishJobStatusResponse(&record.response)
	if queue, ok := m.queues[accountKey(resp.TenantID, resp.AccountID)]; ok {
		stats := queue.snapshot()
		resp.QueueDepth = stats.QueueDepth
		resp.QueueCapacity = stats.QueueCapacity
		resp.ActiveJobs = stats.ActiveJobs
	}
	return resp, nil
}

func (m *PublishJobManager) QueueStats() map[string]AccountJobQueueStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]AccountJobQueueStats, len(m.queues))
	for key, queue := range m.queues {
		result[key] = queue.snapshot()
	}
	return result
}

func (m *PublishJobManager) RuntimeStats() PublishJobRuntimeStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := PublishJobRuntimeStats{
		OutstandingJobs: m.outstandingJobs,
		QueueCount:      len(m.queues),
		MaxOutstanding:  m.maxOutstandingJobs,
	}
	for _, queue := range m.queues {
		snapshot := queue.snapshot()
		stats.QueuedJobs += snapshot.QueueDepth
		stats.ActiveJobs += snapshot.ActiveJobs
	}
	return stats
}

func (m *PublishJobManager) ensureQueue(scope AccountScope) (*accountJobQueue, error) {
	if m.resolveAccount == nil {
		return nil, newAppError("ASYNC_NOT_AVAILABLE", "account resolver unavailable for async publish", 500, false, nil, nil)
	}

	resolved, err := m.resolveAccount(scope)
	if err != nil {
		return nil, classifyError(err)
	}

	key := accountKey(resolved.TenantID, resolved.AccountID)

	m.mu.Lock()
	defer m.mu.Unlock()

	if queue, ok := m.queues[key]; ok {
		return queue, nil
	}

	workerCount := resolved.MaxConcurrency
	if workerCount <= 0 {
		workerCount = 1
	}
	capacity := m.accountQueueCapacity
	if capacity <= 0 {
		capacity = 1
	}

	queue := &accountJobQueue{
		scope:    AccountScope{TenantID: resolved.TenantID, AccountID: resolved.AccountID},
		key:      key,
		tasks:    make(chan *publishJobTask, capacity),
		workers:  workerCount,
		capacity: capacity,
	}
	for i := 0; i < workerCount; i++ {
		go queue.worker(m)
	}
	m.queues[key] = queue
	return queue, nil
}

func (m *PublishJobManager) enqueueJob(jobID string, record *publishJobRecord, queue *accountJobQueue, task *publishJobTask) error {
	m.mu.Lock()
	m.pruneExpiredLocked()
	if m.maxOutstandingJobs > 0 && m.outstandingJobs >= m.maxOutstandingJobs {
		m.mu.Unlock()
		return newAppError("GLOBAL_JOB_QUEUE_FULL", "too many outstanding publish jobs", 429, true, nil, map[string]any{
			"max_outstanding_jobs": m.maxOutstandingJobs,
		})
	}
	m.jobs[jobID] = record
	m.outstandingJobs++
	m.mu.Unlock()

	if err := queue.enqueue(task); err != nil {
		m.mu.Lock()
		delete(m.jobs, jobID)
		if m.outstandingJobs > 0 {
			m.outstandingJobs--
		}
		m.mu.Unlock()
		return err
	}
	return nil
}

func (m *PublishJobManager) runContentJob(jobID string, scope AccountScope, req *PublishRequest) {
	startedAt := m.now().UTC()
	m.markJobRunning(jobID, startedAt)

	ctx, cancel := m.newJobContext(scope, operationMetadata{
		Name:    OperationPublishContent,
		TaskID:  req.TaskID,
		BatchID: req.BatchID,
	}, OperationPublishContent)
	defer cancel()

	result, err := m.runContent(ctx, req)
	m.finishJob(jobID, startedAt, result, err)
}

func (m *PublishJobManager) runVideoJob(jobID string, scope AccountScope, req *PublishVideoRequest) {
	startedAt := m.now().UTC()
	m.markJobRunning(jobID, startedAt)

	ctx, cancel := m.newJobContext(scope, operationMetadata{
		Name:    OperationPublishVideo,
		TaskID:  req.TaskID,
		BatchID: req.BatchID,
	}, OperationPublishVideo)
	defer cancel()

	result, err := m.runVideo(ctx, req)
	m.finishJob(jobID, startedAt, result, err)
}

func (m *PublishJobManager) newJobContext(scope AccountScope, meta operationMetadata, op OperationName) (context.Context, context.CancelFunc) {
	ctx := context.Background()
	ctx = WithAccountScope(ctx, scope)
	ctx = withOperationMetadata(ctx, meta)
	if m.withTimeout == nil {
		return context.WithCancel(ctx)
	}
	return m.withTimeout(ctx, op)
}

func (m *PublishJobManager) markJobRunning(jobID string, startedAt time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, ok := m.jobs[jobID]
	if !ok {
		return
	}
	record.response.Status = PublishJobStatusRunning
	record.response.StartedAt = formatTime(startedAt)
}

func (m *PublishJobManager) finishJob(jobID string, startedAt time.Time, result *PublishExecutionResult, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	record, ok := m.jobs[jobID]
	if !ok {
		return
	}

	finishedAt := m.now().UTC()
	if result == nil {
		result = &PublishExecutionResult{
			OK:             err == nil,
			Mode:           string(PublishModeAsync),
			Operation:      record.response.Operation,
			TaskID:         record.response.TaskID,
			BatchID:        record.response.BatchID,
			TenantID:       record.response.TenantID,
			AccountID:      record.response.AccountID,
			Status:         "unknown",
			PublishStartAt: formatTime(startedAt),
			PublishEndAt:   formatTime(finishedAt),
		}
	}

	result.Mode = string(PublishModeAsync)
	if result.PublishStartAt == "" {
		result.PublishStartAt = formatTime(startedAt)
	}
	if result.PublishEndAt == "" {
		result.PublishEndAt = formatTime(finishedAt)
	}
	if result.DurationMs == 0 {
		result.DurationMs = finishedAt.Sub(startedAt).Milliseconds()
	}

	record.response.Result = clonePublishExecutionResult(result)
	record.response.FinishedAt = formatTime(finishedAt)
	record.response.StartedAt = firstNonEmpty(record.response.StartedAt, formatTime(startedAt))
	record.response.TaskID = firstNonEmpty(record.response.TaskID, result.TaskID)
	record.response.BatchID = firstNonEmpty(record.response.BatchID, result.BatchID)
	record.response.TenantID = firstNonEmpty(record.response.TenantID, result.TenantID)
	record.response.AccountID = firstNonEmpty(record.response.AccountID, result.AccountID)

	if err == nil {
		record.response.Status = PublishJobStatusSucceeded
	} else {
		appErr := classifyError(err)
		if appErr.Code == "OPERATION_CANCELED" {
			record.response.Status = PublishJobStatusCanceled
		} else {
			record.response.Status = PublishJobStatusFailed
		}
	}

	if m.outstandingJobs > 0 {
		m.outstandingJobs--
	}
}

func (m *PublishJobManager) pruneExpiredLocked() {
	if m.ttl <= 0 {
		return
	}
	now := m.now().UTC()
	for jobID, record := range m.jobs {
		if record.response.FinishedAt == "" {
			continue
		}
		finishedAt, err := time.Parse(time.RFC3339, record.response.FinishedAt)
		if err != nil {
			continue
		}
		if now.Sub(finishedAt) > m.ttl {
			delete(m.jobs, jobID)
		}
	}
}

func (m *PublishJobManager) newJobID() string {
	return fmt.Sprintf("job_%d", m.now().UnixNano())
}

func clonePublishRequest(req *PublishRequest) *PublishRequest {
	if req == nil {
		return nil
	}
	copyReq := *req
	copyReq.Images = copyStrings(req.Images)
	copyReq.Tags = copyStrings(req.Tags)
	copyReq.Products = copyStrings(req.Products)
	return &copyReq
}

func clonePublishVideoRequest(req *PublishVideoRequest) *PublishVideoRequest {
	if req == nil {
		return nil
	}
	copyReq := *req
	copyReq.Tags = copyStrings(req.Tags)
	copyReq.Products = copyStrings(req.Products)
	return &copyReq
}

func clonePublishExecutionResult(result *PublishExecutionResult) *PublishExecutionResult {
	if result == nil {
		return nil
	}
	copyResult := *result
	copyResult.ImagePaths = copyStrings(result.ImagePaths)
	copyResult.StagedImagePaths = copyStrings(result.StagedImagePaths)
	return &copyResult
}

func clonePublishJobStatusResponse(resp *PublishJobStatusResponse) *PublishJobStatusResponse {
	if resp == nil {
		return nil
	}
	copyResp := *resp
	copyResp.Result = clonePublishExecutionResult(resp.Result)
	return &copyResp
}

func (q *accountJobQueue) enqueue(task *publishJobTask) error {
	select {
	case q.tasks <- task:
		return nil
	default:
		stats := q.snapshot()
		return newAppError("ACCOUNT_QUEUE_FULL", "account publish queue is full", 429, true, nil, map[string]any{
			"tenant_id":      q.scope.TenantID,
			"account_id":     q.scope.AccountID,
			"queue_depth":    stats.QueueDepth,
			"queue_capacity": stats.QueueCapacity,
		})
	}
}

func (q *accountJobQueue) worker(manager *PublishJobManager) {
	for task := range q.tasks {
		q.markStarted()
		task.run()
		q.markFinished()
	}
}

func (q *accountJobQueue) markStarted() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.activeJobs++
}

func (q *accountJobQueue) markFinished() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.activeJobs > 0 {
		q.activeJobs--
	}
}

func (q *accountJobQueue) snapshot() AccountJobQueueStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return AccountJobQueueStats{
		TenantID:      q.scope.TenantID,
		AccountID:     q.scope.AccountID,
		QueueDepth:    len(q.tasks),
		QueueCapacity: q.capacity,
		ActiveJobs:    q.activeJobs,
		Workers:       q.workers,
	}
}

func isAsyncMode(mode string) bool {
	return strings.EqualFold(strings.TrimSpace(mode), string(PublishModeAsync))
}

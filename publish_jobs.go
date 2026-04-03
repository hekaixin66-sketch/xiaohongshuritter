package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type publishContentExecutor func(context.Context, *PublishRequest) (*PublishExecutionResult, error)
type publishVideoExecutor func(context.Context, *PublishVideoRequest) (*PublishExecutionResult, error)

const (
	defaultPublishJobTimeout         = 10 * time.Minute
	defaultPublishJobTTL             = time.Hour
	defaultPublishJobCleanupInterval = time.Minute
	defaultPublishJobCloseTimeout    = 15 * time.Second

	publishJobStatusAccepted  = "accepted"
	publishJobStatusRunning   = "running"
	publishJobStatusSucceeded = "succeeded"
	publishJobStatusFailed    = "failed"
)

var (
	ErrPublishJobQueueFull       = errors.New("publish async queue is full")
	ErrPublishJobManagerClosed   = errors.New("publish async manager is closed")
	ErrPublishJobNotFound        = errors.New("publish async job not found")
	ErrPublishJobShutdownTimeout = errors.New("publish async manager shutdown timeout")
)

type publishJob struct {
	status    PublishJobStatusResponse
	cleanupAt time.Time
	cancel    context.CancelFunc
}

type publishJobTask struct {
	jobID string
	run   func()
}

type publishJobManagerConfig struct {
	workerCount     int
	queueSize       int
	jobTimeout      time.Duration
	jobTTL          time.Duration
	cleanupInterval time.Duration
	closeTimeout    time.Duration
}

type PublishJobManager struct {
	mu              sync.RWMutex
	jobs            map[string]*publishJob
	now             func() time.Time
	content         publishContentExecutor
	video           publishVideoExecutor
	queue           chan publishJobTask
	jobTimeout      time.Duration
	jobTTL          time.Duration
	cleanupInterval time.Duration
	closeTimeout    time.Duration
	jobSeq          uint64
	backgroundWG    sync.WaitGroup
	closeOnce       sync.Once
	stopCleanup     chan struct{}
	stopWorkers     chan struct{}
	closed          bool
}

func NewPublishJobManager(content publishContentExecutor, video publishVideoExecutor) *PublishJobManager {
	return newPublishJobManager(content, video, loadPublishJobManagerConfigFromEnv())
}

func newPublishJobManager(content publishContentExecutor, video publishVideoExecutor, cfg publishJobManagerConfig) *PublishJobManager {
	cfg = normalizePublishJobManagerConfig(cfg)
	manager := &PublishJobManager{
		jobs:            make(map[string]*publishJob),
		now:             time.Now,
		content:         content,
		video:           video,
		queue:           make(chan publishJobTask, cfg.queueSize),
		jobTimeout:      cfg.jobTimeout,
		jobTTL:          cfg.jobTTL,
		cleanupInterval: cfg.cleanupInterval,
		closeTimeout:    cfg.closeTimeout,
		stopCleanup:     make(chan struct{}),
		stopWorkers:     make(chan struct{}),
	}
	for i := 0; i < cfg.workerCount; i++ {
		manager.backgroundWG.Add(1)
		go manager.runWorker()
	}
	manager.backgroundWG.Add(1)
	go manager.runCleanupLoop()
	return manager
}

func loadPublishJobManagerConfigFromEnv() publishJobManagerConfig {
	workerCount := parsePositiveIntEnv("XHS_PUBLISH_JOB_WORKERS", parsePositiveIntEnv("XHS_MAX_CONCURRENCY", defaultGlobalConcurrency))
	return publishJobManagerConfig{
		workerCount:     workerCount,
		queueSize:       parsePositiveIntEnv("XHS_PUBLISH_JOB_QUEUE_SIZE", workerCount*4),
		jobTimeout:      parsePositiveDurationEnv("XHS_PUBLISH_JOB_TIMEOUT", defaultPublishJobTimeout),
		jobTTL:          parsePositiveDurationEnv("XHS_PUBLISH_JOB_TTL", defaultPublishJobTTL),
		cleanupInterval: parsePositiveDurationEnv("XHS_PUBLISH_JOB_CLEANUP_INTERVAL", defaultPublishJobCleanupInterval),
		closeTimeout:    parsePositiveDurationEnv("XHS_PUBLISH_JOB_CLOSE_TIMEOUT", defaultPublishJobCloseTimeout),
	}
}

func normalizePublishJobManagerConfig(cfg publishJobManagerConfig) publishJobManagerConfig {
	if cfg.workerCount <= 0 {
		cfg.workerCount = 1
	}
	if cfg.queueSize <= 0 {
		cfg.queueSize = cfg.workerCount
	}
	if cfg.jobTimeout <= 0 {
		cfg.jobTimeout = defaultPublishJobTimeout
	}
	if cfg.jobTTL <= 0 {
		cfg.jobTTL = defaultPublishJobTTL
	}
	if cfg.cleanupInterval <= 0 {
		cfg.cleanupInterval = defaultPublishJobCleanupInterval
	}
	if cfg.closeTimeout <= 0 {
		cfg.closeTimeout = defaultPublishJobCloseTimeout
	}
	return cfg
}

func (m *PublishJobManager) SubmitContent(scope AccountScope, req *PublishRequest) (*PublishAcceptedResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("publish request is required")
	}
	if m.content == nil {
		return nil, fmt.Errorf("content publish executor unavailable")
	}

	taskID := ensureTaskID(req.TaskID, "pub")
	reqCopy := *req
	reqCopy.TaskID = taskID
	reqCopy.Mode = string(PublishModeAsync)

	return m.submit(scope, taskID, req.BatchID, func(jobID string) publishJobTask {
		return publishJobTask{
			jobID: jobID,
			run: func() {
				m.runContent(scope, jobID, &reqCopy)
			},
		}
	})
}

func (m *PublishJobManager) SubmitVideo(scope AccountScope, req *PublishVideoRequest) (*PublishAcceptedResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("publish request is required")
	}
	if m.video == nil {
		return nil, fmt.Errorf("video publish executor unavailable")
	}

	taskID := ensureTaskID(req.TaskID, "pub")
	reqCopy := *req
	reqCopy.TaskID = taskID
	reqCopy.Mode = string(PublishModeAsync)

	return m.submit(scope, taskID, req.BatchID, func(jobID string) publishJobTask {
		return publishJobTask{
			jobID: jobID,
			run: func() {
				m.runVideo(scope, jobID, &reqCopy)
			},
		}
	})
}

func (m *PublishJobManager) submit(scope AccountScope, taskID, batchID string, buildTask func(jobID string) publishJobTask) (*PublishAcceptedResponse, error) {
	jobID := m.newJobID()
	createdAt := formatRFC3339(m.now())
	status := newPublishAcceptedStatus(scope, jobID, taskID, batchID, createdAt)
	if err := m.enqueue(status, buildTask(jobID)); err != nil {
		return nil, err
	}
	return newPublishAcceptedResponse(status), nil
}

func newPublishAcceptedStatus(scope AccountScope, jobID, taskID, batchID, createdAt string) PublishJobStatusResponse {
	return PublishJobStatusResponse{
		JobID:     jobID,
		Status:    publishJobStatusAccepted,
		TaskID:    taskID,
		BatchID:   batchID,
		TenantID:  scope.TenantID,
		AccountID: scope.AccountID,
		Mode:      string(PublishModeAsync),
		CreatedAt: createdAt,
	}
}

func newPublishAcceptedResponse(status PublishJobStatusResponse) *PublishAcceptedResponse {
	return &PublishAcceptedResponse{
		Accepted:  true,
		JobID:     status.JobID,
		Status:    status.Status,
		TaskID:    status.TaskID,
		BatchID:   status.BatchID,
		TenantID:  status.TenantID,
		AccountID: status.AccountID,
		Mode:      status.Mode,
		CreatedAt: status.CreatedAt,
	}
}

func (m *PublishJobManager) Get(jobID string) (*PublishJobStatusResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	trimmedJobID := strings.TrimSpace(jobID)
	job, ok := m.jobs[trimmedJobID]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrPublishJobNotFound, trimmedJobID)
	}

	copyStatus := job.status
	if job.status.Result != nil {
		copyResult := *job.status.Result
		copyStatus.Result = &copyResult
	}

	return &copyStatus, nil
}

func (m *PublishJobManager) enqueue(status PublishJobStatusResponse, task publishJobTask) error {
	if task.run == nil {
		return fmt.Errorf("publish job task is invalid")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrPublishJobManagerClosed
	}
	m.jobs[status.JobID] = &publishJob{status: status}

	select {
	case m.queue <- task:
		return nil
	default:
		delete(m.jobs, status.JobID)
		return fmt.Errorf("%w: capacity=%d", ErrPublishJobQueueFull, cap(m.queue))
	}
}

func (m *PublishJobManager) runWorker() {
	defer m.backgroundWG.Done()

	for {
		select {
		case <-m.stopWorkers:
			return
		case task := <-m.queue:
			if m.isClosed() {
				m.markFinished(task.jobID, m.now(), nil, ErrPublishJobManagerClosed)
				continue
			}
			select {
			case <-m.stopWorkers:
				m.markFinished(task.jobID, m.now(), nil, ErrPublishJobManagerClosed)
				continue
			default:
			}
			if task.run != nil {
				task.run()
				continue
			}
			started := m.now()
			m.markRunning(task.jobID, started)
			m.markFinished(task.jobID, started, nil, fmt.Errorf("publish job task is invalid"))
		}
	}
}

func (m *PublishJobManager) runCleanupLoop() {
	defer m.backgroundWG.Done()

	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanupExpiredJobs()
		case <-m.stopCleanup:
			return
		}
	}
}

func (m *PublishJobManager) Close() {
	if m == nil {
		return
	}
	m.closeOnce.Do(func() {
		m.mu.Lock()
		m.closed = true
		close(m.stopCleanup)
		close(m.stopWorkers)
		m.mu.Unlock()

		done := make(chan struct{})
		go func() {
			m.backgroundWG.Wait()
			close(done)
		}()

		select {
		case <-done:
			m.failPendingJobs(ErrPublishJobManagerClosed)
		case <-time.After(m.closeTimeout):
			m.cancelRunningJobs()
			m.failPendingJobs(ErrPublishJobShutdownTimeout)
		}
	})
}

func (m *PublishJobManager) ensureResultLocked(jobID string, job *publishJob) *PublishExecutionResult {
	if job == nil {
		return nil
	}
	if job.status.Result == nil {
		job.status.Result = &PublishExecutionResult{
			JobID:     jobID,
			TaskID:    job.status.TaskID,
			BatchID:   job.status.BatchID,
			TenantID:  job.status.TenantID,
			AccountID: job.status.AccountID,
			Mode:      string(PublishModeAsync),
		}
	}
	job.status.Result.JobID = jobID
	job.status.Result.Mode = string(PublishModeAsync)
	return job.status.Result
}

func (m *PublishJobManager) failPendingJobs(cause error) {
	finished := m.now()
	finishedAt := formatRFC3339(finished)

	m.mu.Lock()
	defer m.mu.Unlock()

	for jobID, job := range m.jobs {
		if job == nil || job.status.FinishedAt != "" {
			continue
		}
		job.status.Status = publishJobStatusFailed
		job.status.FinishedAt = finishedAt
		job.cleanupAt = finished.Add(m.jobTTL)
		job.cancel = nil
		result := m.ensureResultLocked(jobID, job)
		result.OK = false
		result.Status = publishJobStatusFailed
		if result.ErrorCode == "" {
			result.ErrorCode = shutdownErrorCode(cause)
		}
		if result.ErrorMessage == "" {
			result.ErrorMessage = cause.Error()
		}
	}
}

func (m *PublishJobManager) cancelRunningJobs() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, job := range m.jobs {
		if job == nil || job.status.FinishedAt != "" || job.cancel == nil {
			continue
		}
		job.cancel()
	}
}

func (m *PublishJobManager) cleanupExpiredJobs() int {
	now := m.now()
	m.mu.Lock()
	defer m.mu.Unlock()

	removed := 0
	for jobID, job := range m.jobs {
		if job.cleanupAt.IsZero() || now.Before(job.cleanupAt) {
			continue
		}
		delete(m.jobs, jobID)
		removed++
	}
	return removed
}

func (m *PublishJobManager) runContent(scope AccountScope, jobID string, req *PublishRequest) {
	started := m.now()
	ctx, cancel := context.WithTimeout(WithAccountScope(context.Background(), scope), m.jobTimeout)
	defer cancel()

	m.markRunningWithCancel(jobID, started, cancel)
	result, err := m.content(ctx, req)
	m.markFinished(jobID, started, result, err)
}

func (m *PublishJobManager) runVideo(scope AccountScope, jobID string, req *PublishVideoRequest) {
	started := m.now()
	ctx, cancel := context.WithTimeout(WithAccountScope(context.Background(), scope), m.jobTimeout)
	defer cancel()

	m.markRunningWithCancel(jobID, started, cancel)
	result, err := m.video(ctx, req)
	m.markFinished(jobID, started, result, err)
}

func (m *PublishJobManager) markRunning(jobID string, started time.Time) {
	m.markRunningWithCancel(jobID, started, nil)
}

func (m *PublishJobManager) markRunningWithCancel(jobID string, started time.Time, cancel context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return
	}
	if job.status.FinishedAt != "" {
		if cancel != nil {
			cancel()
		}
		return
	}
	job.status.Status = publishJobStatusRunning
	job.status.StartedAt = formatRFC3339(started)
	job.cancel = cancel
}

func (m *PublishJobManager) markFinished(jobID string, started time.Time, result *PublishExecutionResult, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, ok := m.jobs[jobID]
	if !ok {
		return
	}
	if job.status.FinishedAt != "" {
		job.cancel = nil
		return
	}

	finished := m.now()
	job.status.StartedAt = firstNonEmpty(job.status.StartedAt, formatRFC3339(started))
	job.status.FinishedAt = formatRFC3339(finished)
	job.cleanupAt = finished.Add(m.jobTTL)
	job.cancel = nil

	if result != nil {
		resultCopy := *result
		resultCopy.JobID = jobID
		resultCopy.Mode = string(PublishModeAsync)
		job.status.Result = &resultCopy
	}

	if err != nil {
		job.status.Status = publishJobStatusFailed
		result := m.ensureResultLocked(jobID, job)
		if result.ErrorCode == "" {
			if errors.Is(err, ErrPublishJobManagerClosed) {
				result.ErrorCode = "PUBLISH_JOB_MANAGER_CLOSED"
			} else {
				result.ErrorCode = "PUBLISH_FAILED"
			}
		}
		if result.ErrorMessage == "" {
			result.ErrorMessage = err.Error()
		}
		result.OK = false
		result.Status = publishJobStatusFailed
		return
	}

	job.status.Status = publishJobStatusSucceeded
	if job.status.Result != nil {
		job.status.Result.JobID = jobID
		job.status.Result.Mode = string(PublishModeAsync)
		job.status.Result.Status = publishJobStatusSucceeded
		job.status.Result.OK = true
	}
}

func (m *PublishJobManager) newJobID() string {
	seq := atomic.AddUint64(&m.jobSeq, 1)
	return fmt.Sprintf("xhs_%d_%d", m.now().UnixNano(), seq)
}

func (m *PublishJobManager) isClosed() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.closed
}

func shutdownErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrPublishJobShutdownTimeout):
		return "PUBLISH_JOB_SHUTDOWN_TIMEOUT"
	default:
		return "PUBLISH_JOB_MANAGER_CLOSED"
	}
}

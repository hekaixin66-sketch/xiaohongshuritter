package main

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type PublishMetrics struct {
	Total             int64  `json:"total"`
	Success           int64  `json:"success"`
	Failure           int64  `json:"failure"`
	AverageDurationMs int64  `json:"average_duration_ms"`
	LastDurationMs    int64  `json:"last_duration_ms"`
	LastStartedAt     string `json:"last_started_at,omitempty"`
	LastFinishedAt    string `json:"last_finished_at,omitempty"`
}

type BrowserMetrics struct {
	ActiveSessions int64  `json:"active_sessions"`
	MaxSessions    int64  `json:"max_sessions"`
	ProcessCount   int64  `json:"process_count"`
	LastCleanupAt  string `json:"last_cleanup_at,omitempty"`
	JanitorRuns    int64  `json:"janitor_runs"`
	JanitorKills   int64  `json:"janitor_kills"`
}

type ProcessMetrics struct {
	Goroutines int    `json:"goroutines"`
	HeapAlloc  uint64 `json:"heap_alloc_bytes"`
}

type RuntimeStats struct {
	Publish PublishMetrics `json:"publish"`
	Browser BrowserMetrics `json:"browser"`
	Process ProcessMetrics `json:"process"`
}

type ServiceMetrics struct {
	publishTotal          atomic.Int64
	publishSuccess        atomic.Int64
	publishFailure        atomic.Int64
	totalPublishDuration  atomic.Int64
	lastPublishDurationMs atomic.Int64
	activeBrowserSessions atomic.Int64
	maxBrowserSessions    atomic.Int64
	browserProcessCount   atomic.Int64
	browserJanitorRuns    atomic.Int64
	browserJanitorKills   atomic.Int64

	mu             sync.Mutex
	lastStartedAt  time.Time
	lastFinishedAt time.Time
	lastCleanupAt  time.Time
}

func NewServiceMetrics() *ServiceMetrics {
	return &ServiceMetrics{}
}

func (m *ServiceMetrics) RecordPublish(startedAt time.Time, duration time.Duration, success bool) {
	m.publishTotal.Add(1)
	if success {
		m.publishSuccess.Add(1)
	} else {
		m.publishFailure.Add(1)
	}
	m.totalPublishDuration.Add(duration.Nanoseconds())
	m.lastPublishDurationMs.Store(duration.Milliseconds())

	m.mu.Lock()
	m.lastStartedAt = startedAt
	m.lastFinishedAt = startedAt.Add(duration)
	m.mu.Unlock()
}

func (m *ServiceMetrics) BrowserSessionStarted() {
	m.activeBrowserSessions.Add(1)
}

func (m *ServiceMetrics) BrowserSessionEnded() {
	m.activeBrowserSessions.Add(-1)
}

func (m *ServiceMetrics) ActiveBrowserSessions() int64 {
	return m.activeBrowserSessions.Load()
}

func (m *ServiceMetrics) RecordBrowserProcessSnapshot(count int, maxSessions int) {
	m.browserProcessCount.Store(int64(count))
	if maxSessions > 0 {
		m.maxBrowserSessions.Store(int64(maxSessions))
	}
}

func (m *ServiceMetrics) RecordBrowserJanitorRun(at time.Time, killed int) {
	m.browserJanitorRuns.Add(1)
	m.browserJanitorKills.Add(int64(killed))
	m.mu.Lock()
	m.lastCleanupAt = at
	m.mu.Unlock()
}

func (m *ServiceMetrics) Snapshot() RuntimeStats {
	total := m.publishTotal.Load()
	totalDuration := m.totalPublishDuration.Load()
	var avgMs int64
	if total > 0 {
		avgMs = time.Duration(totalDuration / total).Milliseconds()
	}

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	m.mu.Lock()
	lastStartedAt := m.lastStartedAt
	lastFinishedAt := m.lastFinishedAt
	lastCleanupAt := m.lastCleanupAt
	m.mu.Unlock()

	return RuntimeStats{
		Publish: PublishMetrics{
			Total:             total,
			Success:           m.publishSuccess.Load(),
			Failure:           m.publishFailure.Load(),
			AverageDurationMs: avgMs,
			LastDurationMs:    m.lastPublishDurationMs.Load(),
			LastStartedAt:     formatTime(lastStartedAt),
			LastFinishedAt:    formatTime(lastFinishedAt),
		},
		Browser: BrowserMetrics{
			ActiveSessions: m.activeBrowserSessions.Load(),
			MaxSessions:    m.maxBrowserSessions.Load(),
			ProcessCount:   m.browserProcessCount.Load(),
			LastCleanupAt:  formatTime(lastCleanupAt),
			JanitorRuns:    m.browserJanitorRuns.Load(),
			JanitorKills:   m.browserJanitorKills.Load(),
		},
		Process: ProcessMetrics{
			Goroutines: runtime.NumGoroutine(),
			HeapAlloc:  mem.HeapAlloc,
		},
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

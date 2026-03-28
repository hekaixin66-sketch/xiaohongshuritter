package main

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type BrowserGuard struct {
	sem              chan struct{}
	cleanupInterval  time.Duration
	processWarnLimit int
	processKillLimit int
	metrics          *ServiceMetrics
}

func NewBrowserGuard(maxSessions int, metrics *ServiceMetrics) *BrowserGuard {
	maxSessions = parsePositiveIntEnv("XHS_MAX_BROWSER_SESSIONS", maxSessions)
	if maxSessions <= 0 {
		maxSessions = 1
	}

	warnLimit := parsePositiveIntEnv("XHS_BROWSER_PROCESS_WARN_LIMIT", maxSessions*6)
	killLimit := parsePositiveIntEnv("XHS_BROWSER_PROCESS_KILL_LIMIT", maxSessions*8)
	interval := parsePositiveDurationEnv("XHS_BROWSER_JANITOR_INTERVAL", 2*time.Minute)

	guard := &BrowserGuard{
		sem:              make(chan struct{}, maxSessions),
		cleanupInterval:  interval,
		processWarnLimit: warnLimit,
		processKillLimit: killLimit,
		metrics:          metrics,
	}
	if metrics != nil {
		metrics.RecordBrowserProcessSnapshot(0, maxSessions)
	}
	return guard
}

func (g *BrowserGuard) MaxSessions() int {
	if g == nil {
		return 0
	}
	return cap(g.sem)
}

func (g *BrowserGuard) Acquire(ctx context.Context) (func(), error) {
	if g == nil || g.sem == nil {
		return func() {}, nil
	}

	select {
	case g.sem <- struct{}{}:
		return func() {
			select {
			case <-g.sem:
			default:
			}
		}, nil
	case <-ctx.Done():
		return nil, newAppError("BROWSER_SESSION_LIMIT_REACHED", "browser session limit reached", 429, true, ctx.Err(), nil)
	}
}

func (g *BrowserGuard) Start(ctx context.Context, activeSessionsFn func() int64) {
	if g == nil || g.cleanupInterval <= 0 {
		return
	}

	ticker := time.NewTicker(g.cleanupInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				g.runJanitor(activeSessionsFn)
			}
		}
	}()
}

func (g *BrowserGuard) runJanitor(activeSessionsFn func() int64) {
	processes := listBrowserProcesses()
	g.metrics.RecordBrowserProcessSnapshot(len(processes), g.MaxSessions())
	if len(processes) == 0 {
		return
	}

	activeSessions := int64(0)
	if activeSessionsFn != nil {
		activeSessions = activeSessionsFn()
	}

	if len(processes) >= g.processWarnLimit {
		g.metrics.RecordBrowserJanitorRun(time.Now().UTC(), 0)
	}

	if activeSessions > 0 || len(processes) < g.processKillLimit {
		return
	}

	killed := 0
	for _, process := range processes {
		if process.pid <= 0 {
			continue
		}
		proc, err := os.FindProcess(process.pid)
		if err != nil {
			continue
		}
		if err := proc.Kill(); err == nil {
			killed++
		}
	}
	g.metrics.RecordBrowserJanitorRun(time.Now().UTC(), killed)
}

type browserProcess struct {
	pid  int
	name string
}

func listBrowserProcesses() []browserProcess {
	cmd := exec.Command("ps", "-Ao", "pid=,comm=")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(string(output), "\n")
	processes := make([]browserProcess, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		name := strings.ToLower(strings.Join(fields[1:], " "))
		if !isBrowserProcessName(name) {
			continue
		}
		processes = append(processes, browserProcess{pid: pid, name: name})
	}

	return processes
}

func isBrowserProcessName(name string) bool {
	switch runtime.GOOS {
	case "windows":
		return strings.Contains(name, "chrome")
	default:
		return strings.Contains(name, "chromium") ||
			strings.Contains(name, "chrome") ||
			strings.Contains(name, "headless")
	}
}

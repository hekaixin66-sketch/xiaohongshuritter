package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hekaixin66-sketch/xiaohongshuritter/configs"
	"github.com/hekaixin66-sketch/xiaohongshuritter/cookies"
	"github.com/sirupsen/logrus"
)

const (
	defaultAccountConfigPath   = "configs/accounts.json"
	defaultTenantID            = "default"
	defaultAccountID           = "default"
	defaultGlobalConcurrency   = 8
	defaultPerAccountSemaphore = 2
	defaultAcquireTimeout      = 120 * time.Second
	defaultCooldownFailures    = 3
	defaultCooldownDuration    = 10 * time.Minute
)

type EnterpriseAccountConfig struct {
	DefaultTenant         string                   `json:"default_tenant"`
	DefaultAccount        string                   `json:"default_account"`
	GlobalMaxConcurrency  int                      `json:"global_max_concurrency"`
	AcquireTimeoutSeconds int                      `json:"acquire_timeout_seconds,omitempty"`
	Tenants               []EnterpriseTenantConfig `json:"tenants"`
}

type EnterpriseTenantConfig struct {
	ID             string                    `json:"id"`
	Name           string                    `json:"name,omitempty"`
	DefaultAccount string                    `json:"default_account,omitempty"`
	Accounts       []EnterpriseProfileConfig `json:"accounts"`
}

type EnterpriseProfileConfig struct {
	ID             string `json:"id"`
	Name           string `json:"name,omitempty"`
	CookiePath     string `json:"cookie_path,omitempty"`
	MaxConcurrency int    `json:"max_concurrency,omitempty"`
	Headless       *bool  `json:"headless,omitempty"`
	BrowserBin     string `json:"browser_bin,omitempty"`
	Proxy          string `json:"proxy,omitempty"`
}

type AccountInfo struct {
	TenantID            string `json:"tenant_id"`
	TenantName          string `json:"tenant_name,omitempty"`
	AccountID           string `json:"account_id"`
	AccountName         string `json:"account_name,omitempty"`
	DefaultTenant       bool   `json:"default_tenant"`
	DefaultAccount      bool   `json:"default_account"`
	CookiePath          string `json:"cookie_path"`
	MaxConcurrency      int    `json:"max_concurrency"`
	CurrentInFlight     int    `json:"current_in_flight"`
	Headless            bool   `json:"headless"`
	BrowserBin          string `json:"browser_bin,omitempty"`
	Proxy               string `json:"proxy,omitempty"`
	LastSuccessAt       string `json:"last_success_at,omitempty"`
	LastErrorAt         string `json:"last_error_at,omitempty"`
	LastErrorCode       string `json:"last_error_code,omitempty"`
	LastErrorMessage    string `json:"last_error_message,omitempty"`
	ConsecutiveFailures int    `json:"consecutive_failures,omitempty"`
	CooldownUntil       string `json:"cooldown_until,omitempty"`
	QueueDepth          int    `json:"queue_depth,omitempty"`
	QueueCapacity       int    `json:"queue_capacity,omitempty"`
	ActiveJobs          int    `json:"active_jobs,omitempty"`
	QueueWorkers        int    `json:"queue_workers,omitempty"`
}

type ResolvedAccount struct {
	TenantID       string
	TenantName     string
	AccountID      string
	AccountName    string
	CookiePath     string
	MaxConcurrency int
	Headless       bool
	BrowserBin     string
	Proxy          string
}

type accountRuntime struct {
	resolved ResolvedAccount
	sem      chan struct{}
	mu       sync.Mutex

	lastSuccessAt       time.Time
	lastErrorAt         time.Time
	lastErrorCode       string
	lastErrorMessage    string
	consecutiveFailures int
	cooldownUntil       time.Time
}

type AccountManager struct {
	configPath      string
	defaultTenantID string
	defaultAccount  string
	globalSem       chan struct{}
	acquireTimeout  time.Duration
	cooldownAfter   int
	cooldownPeriod  time.Duration

	tenantDefaults map[string]string
	runtimes       map[string]*accountRuntime
}

type AccountSession struct {
	ResolvedAccount
	releaseOnce sync.Once
	releaseFn   func()
}

func (s *AccountSession) Release() {
	if s == nil {
		return
	}
	s.releaseOnce.Do(func() {
		if s.releaseFn != nil {
			s.releaseFn()
		}
	})
}

func NewAccountManagerFromEnv() (*AccountManager, error) {
	configPath := strings.TrimSpace(os.Getenv("XHS_ACCOUNT_CONFIG"))
	explicitConfigPath := configPath != ""
	if configPath == "" {
		configPath = defaultAccountConfigPath
	}

	defaultGlobal := parsePositiveIntEnv("XHS_MAX_CONCURRENCY", defaultGlobalConcurrency)
	defaultPerAccount := parsePositiveIntEnv("XHS_ACCOUNT_MAX_CONCURRENCY", defaultPerAccountSemaphore)
	acquireTimeout := parsePositiveDurationEnv("XHS_ACQUIRE_TIMEOUT", defaultAcquireTimeout)
	cooldownAfter := parsePositiveIntEnv("XHS_ACCOUNT_COOLDOWN_FAILURES", defaultCooldownFailures)
	cooldownDuration := parsePositiveDurationEnv("XHS_ACCOUNT_COOLDOWN_DURATION", defaultCooldownDuration)

	cfg, err := loadEnterpriseConfig(configPath, explicitConfigPath, defaultGlobal, defaultPerAccount)
	if err != nil {
		return nil, err
	}
	if cfg.AcquireTimeoutSeconds > 0 {
		acquireTimeout = time.Duration(cfg.AcquireTimeoutSeconds) * time.Second
	}

	manager, err := buildAccountManager(configPath, cfg, defaultPerAccount)
	if err != nil {
		return nil, err
	}
	manager.acquireTimeout = acquireTimeout
	manager.cooldownAfter = cooldownAfter
	manager.cooldownPeriod = cooldownDuration

	logrus.WithFields(logrus.Fields{
		"config_path":          configPath,
		"default_tenant":       manager.defaultTenantID,
		"default_account":      manager.defaultAccount,
		"global_concurrency":   cap(manager.globalSem),
		"account_concurrency":  defaultPerAccount,
		"account_runtime_size": len(manager.runtimes),
		"cooldown_after":       manager.cooldownAfter,
		"cooldown_period":      manager.cooldownPeriod.String(),
	}).Info("account manager initialized")

	return manager, nil
}

func (m *AccountManager) Resolve(scope AccountScope) (ResolvedAccount, error) {
	tenantID := strings.TrimSpace(scope.TenantID)
	accountID := strings.TrimSpace(scope.AccountID)

	if tenantID == "" {
		tenantID = m.defaultTenantID
	}

	if accountID == "" {
		if tenantID == m.defaultTenantID && m.defaultAccount != "" {
			accountID = m.defaultAccount
		} else {
			accountID = m.tenantDefaults[tenantID]
		}
	}

	key := accountKey(tenantID, accountID)
	runtime, ok := m.runtimes[key]
	if !ok {
		return ResolvedAccount{}, fmt.Errorf("account not found for tenant=%q account=%q", tenantID, accountID)
	}

	return runtime.resolved, nil
}

func (m *AccountManager) Acquire(ctx context.Context, scope AccountScope) (*AccountSession, error) {
	resolved, err := m.Resolve(scope)
	if err != nil {
		return nil, err
	}

	runtime, ok := m.runtimes[accountKey(resolved.TenantID, resolved.AccountID)]
	if !ok {
		return nil, fmt.Errorf("runtime missing for tenant=%q account=%q", resolved.TenantID, resolved.AccountID)
	}
	if err := runtime.checkCooldown(); err != nil {
		return nil, err
	}

	acquireCtx := ctx
	var cancel context.CancelFunc
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && m.acquireTimeout > 0 {
		acquireCtx, cancel = context.WithTimeout(ctx, m.acquireTimeout)
	} else {
		cancel = func() {}
	}
	defer cancel()

	if err := acquireSemaphore(acquireCtx, m.globalSem); err != nil {
		return nil, fmt.Errorf("global concurrency exhausted: %w", err)
	}

	if err := acquireSemaphore(acquireCtx, runtime.sem); err != nil {
		releaseSemaphore(m.globalSem)
		return nil, fmt.Errorf("account concurrency exhausted for tenant=%q account=%q: %w", resolved.TenantID, resolved.AccountID, err)
	}

	return &AccountSession{
		ResolvedAccount: resolved,
		releaseFn: func() {
			releaseSemaphore(runtime.sem)
			releaseSemaphore(m.globalSem)
		},
	}, nil
}

func (m *AccountManager) ListAccounts() []AccountInfo {
	result := make([]AccountInfo, 0, len(m.runtimes))
	for _, runtime := range m.runtimes {
		resolved := runtime.resolved
		stats := runtime.snapshot()
		isDefaultTenant := resolved.TenantID == m.defaultTenantID
		isDefaultAccount := isDefaultTenant && resolved.AccountID == m.defaultAccount

		result = append(result, AccountInfo{
			TenantID:            resolved.TenantID,
			TenantName:          resolved.TenantName,
			AccountID:           resolved.AccountID,
			AccountName:         resolved.AccountName,
			DefaultTenant:       isDefaultTenant,
			DefaultAccount:      isDefaultAccount,
			CookiePath:          resolved.CookiePath,
			MaxConcurrency:      resolved.MaxConcurrency,
			CurrentInFlight:     len(runtime.sem),
			Headless:            resolved.Headless,
			BrowserBin:          resolved.BrowserBin,
			Proxy:               maskCredentials(resolved.Proxy),
			LastSuccessAt:       formatTime(stats.LastSuccessAt),
			LastErrorAt:         formatTime(stats.LastErrorAt),
			LastErrorCode:       stats.LastErrorCode,
			LastErrorMessage:    stats.LastErrorMessage,
			ConsecutiveFailures: stats.ConsecutiveFailures,
			CooldownUntil:       formatTime(stats.CooldownUntil),
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].TenantID == result[j].TenantID {
			return result[i].AccountID < result[j].AccountID
		}
		return result[i].TenantID < result[j].TenantID
	})

	return result
}

func (m *AccountManager) ConfigPath() string {
	return m.configPath
}

func (m *AccountManager) GlobalInFlight() int {
	return len(m.globalSem)
}

func (m *AccountManager) GlobalConcurrencyLimit() int {
	if m == nil || m.globalSem == nil {
		return 0
	}
	return cap(m.globalSem)
}

func (m *AccountManager) RecordPublishResult(scope AccountScope, appErr *AppError) {
	resolved, err := m.Resolve(scope)
	if err != nil {
		return
	}
	runtime, ok := m.runtimes[accountKey(resolved.TenantID, resolved.AccountID)]
	if !ok {
		return
	}

	runtime.mu.Lock()
	defer runtime.mu.Unlock()

	now := time.Now().UTC()
	if appErr == nil {
		runtime.lastSuccessAt = now
		runtime.lastErrorCode = ""
		runtime.lastErrorMessage = ""
		runtime.consecutiveFailures = 0
		runtime.cooldownUntil = time.Time{}
		return
	}

	runtime.lastErrorAt = now
	runtime.lastErrorCode = appErr.Code
	runtime.lastErrorMessage = appErr.Error()
	if !shouldCountTowardsCooldown(appErr) {
		return
	}

	runtime.consecutiveFailures++
	if m.cooldownAfter > 0 && runtime.consecutiveFailures >= m.cooldownAfter && m.cooldownPeriod > 0 {
		runtime.cooldownUntil = now.Add(m.cooldownPeriod)
		logrus.WithFields(logrus.Fields{
			"tenant_id":      resolved.TenantID,
			"account_id":     resolved.AccountID,
			"cooldown_until": runtime.cooldownUntil.Format(time.RFC3339),
			"error_code":     appErr.Code,
		}).Warn("account entered cooldown after repeated failures")
	}
}

func buildAccountManager(path string, cfg EnterpriseAccountConfig, defaultPerAccount int) (*AccountManager, error) {
	if len(cfg.Tenants) == 0 {
		return nil, fmt.Errorf("no tenants configured in %s", path)
	}

	if cfg.GlobalMaxConcurrency <= 0 {
		cfg.GlobalMaxConcurrency = defaultGlobalConcurrency
	}

	manager := &AccountManager{
		configPath:     path,
		globalSem:      make(chan struct{}, cfg.GlobalMaxConcurrency),
		tenantDefaults: make(map[string]string),
		runtimes:       make(map[string]*accountRuntime),
	}

	for tenantIdx := range cfg.Tenants {
		tenant := cfg.Tenants[tenantIdx]
		tenant.ID = strings.TrimSpace(tenant.ID)
		if tenant.ID == "" {
			return nil, fmt.Errorf("tenant[%d] id is required", tenantIdx)
		}
		if len(tenant.Accounts) == 0 {
			return nil, fmt.Errorf("tenant %q has no accounts", tenant.ID)
		}

		if strings.TrimSpace(tenant.DefaultAccount) == "" {
			tenant.DefaultAccount = tenant.Accounts[0].ID
		}
		manager.tenantDefaults[tenant.ID] = tenant.DefaultAccount

		for accountIdx := range tenant.Accounts {
			profile := tenant.Accounts[accountIdx]
			profile.ID = strings.TrimSpace(profile.ID)
			if profile.ID == "" {
				return nil, fmt.Errorf("tenant %q account[%d] id is required", tenant.ID, accountIdx)
			}

			resolved := resolveProfileConfig(tenant, profile, defaultPerAccount)
			key := accountKey(resolved.TenantID, resolved.AccountID)
			if _, exists := manager.runtimes[key]; exists {
				return nil, fmt.Errorf("duplicate account key %q", key)
			}
			manager.runtimes[key] = &accountRuntime{
				resolved: resolved,
				sem:      make(chan struct{}, resolved.MaxConcurrency),
			}
		}
	}

	if strings.TrimSpace(cfg.DefaultTenant) == "" {
		cfg.DefaultTenant = cfg.Tenants[0].ID
	}
	manager.defaultTenantID = cfg.DefaultTenant

	if _, ok := manager.tenantDefaults[manager.defaultTenantID]; !ok {
		return nil, fmt.Errorf("default tenant %q not found", manager.defaultTenantID)
	}

	if strings.TrimSpace(cfg.DefaultAccount) == "" {
		cfg.DefaultAccount = manager.tenantDefaults[manager.defaultTenantID]
	}
	manager.defaultAccount = cfg.DefaultAccount

	if _, ok := manager.runtimes[accountKey(manager.defaultTenantID, manager.defaultAccount)]; !ok {
		return nil, fmt.Errorf("default account %q not found in default tenant %q", manager.defaultAccount, manager.defaultTenantID)
	}

	return manager, nil
}

func loadEnterpriseConfig(path string, explicit bool, defaultGlobal int, defaultPerAccount int) (EnterpriseAccountConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && !explicit {
			return defaultEnterpriseConfig(defaultGlobal, defaultPerAccount), nil
		}
		return EnterpriseAccountConfig{}, fmt.Errorf("failed to read account config file %s: %w", path, err)
	}

	var cfg EnterpriseAccountConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return EnterpriseAccountConfig{}, fmt.Errorf("failed to parse account config file %s: %w", path, err)
	}

	return cfg, nil
}

func defaultEnterpriseConfig(defaultGlobal int, defaultPerAccount int) EnterpriseAccountConfig {
	headless := configs.IsHeadless()
	return EnterpriseAccountConfig{
		DefaultTenant:        defaultTenantID,
		DefaultAccount:       defaultAccountID,
		GlobalMaxConcurrency: defaultGlobal,
		Tenants: []EnterpriseTenantConfig{
			{
				ID:             defaultTenantID,
				Name:           "Default Tenant",
				DefaultAccount: defaultAccountID,
				Accounts: []EnterpriseProfileConfig{
					{
						ID:             defaultAccountID,
						Name:           "Default Account",
						CookiePath:     cookies.GetCookiesFilePath(),
						MaxConcurrency: defaultPerAccount,
						Headless:       &headless,
						BrowserBin:     configs.GetBinPath(),
						Proxy:          os.Getenv("XHS_PROXY"),
					},
				},
			},
		},
	}
}

func resolveProfileConfig(tenant EnterpriseTenantConfig, profile EnterpriseProfileConfig, defaultPerAccount int) ResolvedAccount {
	cookiePath := strings.TrimSpace(profile.CookiePath)
	if cookiePath == "" {
		cookiePath = filepath.Join("data", tenant.ID, profile.ID, "cookies.json")
	}

	headless := configs.IsHeadless()
	if profile.Headless != nil {
		headless = *profile.Headless
	}

	maxConcurrency := profile.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = defaultPerAccount
	}
	if maxConcurrency <= 0 {
		maxConcurrency = 1
	}

	browserBin := strings.TrimSpace(profile.BrowserBin)
	if browserBin == "" {
		browserBin = strings.TrimSpace(configs.GetBinPath())
	}

	proxy := strings.TrimSpace(profile.Proxy)
	if proxy == "" {
		proxy = strings.TrimSpace(os.Getenv("XHS_PROXY"))
	}

	return ResolvedAccount{
		TenantID:       tenant.ID,
		TenantName:     tenant.Name,
		AccountID:      profile.ID,
		AccountName:    profile.Name,
		CookiePath:     cookiePath,
		MaxConcurrency: maxConcurrency,
		Headless:       headless,
		BrowserBin:     browserBin,
		Proxy:          proxy,
	}
}

func parsePositiveIntEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		logrus.WithFields(logrus.Fields{
			"key":      key,
			"value":    raw,
			"fallback": fallback,
		}).Warn("invalid positive int env, fallback applied")
		return fallback
	}
	return parsed
}

func parsePositiveDurationEnv(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		logrus.WithFields(logrus.Fields{
			"key":      key,
			"value":    raw,
			"fallback": fallback.String(),
		}).Warn("invalid duration env, fallback applied")
		return fallback
	}
	return parsed
}

func accountKey(tenantID, accountID string) string {
	return tenantID + "/" + accountID
}

func acquireSemaphore(ctx context.Context, sem chan struct{}) error {
	select {
	case sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseSemaphore(sem chan struct{}) {
	select {
	case <-sem:
	default:
	}
}

func maskCredentials(proxy string) string {
	if proxy == "" {
		return ""
	}
	at := strings.LastIndex(proxy, "@")
	scheme := strings.Index(proxy, "://")
	if at == -1 || scheme == -1 || at < scheme+3 {
		return proxy
	}
	return proxy[:scheme+3] + "***:***@" + proxy[at+1:]
}

func (r *accountRuntime) checkCooldown() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.cooldownUntil.IsZero() {
		return nil
	}
	if time.Now().UTC().After(r.cooldownUntil) {
		r.cooldownUntil = time.Time{}
		r.consecutiveFailures = 0
		return nil
	}

	return newAppError(
		"ACCOUNT_COOLDOWN",
		"account is cooling down",
		429,
		true,
		nil,
		map[string]any{
			"tenant_id":      r.resolved.TenantID,
			"account_id":     r.resolved.AccountID,
			"cooldown_until": r.cooldownUntil.Format(time.RFC3339),
		},
	)
}

func shouldCountTowardsCooldown(err *AppError) bool {
	if err == nil {
		return false
	}
	if !err.Retryable {
		return false
	}
	return err.StatusCode >= 429 || err.StatusCode >= 500
}

type accountRuntimeSnapshot struct {
	LastSuccessAt       time.Time
	LastErrorAt         time.Time
	LastErrorCode       string
	LastErrorMessage    string
	ConsecutiveFailures int
	CooldownUntil       time.Time
}

func (r *accountRuntime) snapshot() accountRuntimeSnapshot {
	r.mu.Lock()
	defer r.mu.Unlock()

	return accountRuntimeSnapshot{
		LastSuccessAt:       r.lastSuccessAt,
		LastErrorAt:         r.lastErrorAt,
		LastErrorCode:       r.lastErrorCode,
		LastErrorMessage:    r.lastErrorMessage,
		ConsecutiveFailures: r.consecutiveFailures,
		CooldownUntil:       r.cooldownUntil,
	}
}

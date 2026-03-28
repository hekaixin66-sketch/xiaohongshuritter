package main

import (
	"os"
	"sort"
	"strings"
	"time"
)

func (s *AppServer) RecommendAccountsForPublish(req SchedulerRecommendationRequest) SchedulerRecommendationResponse {
	queueStats := s.jobManager.QueueStats()
	accounts := decorateAccountInfosWithQueueStats(s.xiaohongshuService.ListAccounts(), queueStats)

	requireCookie := true
	if req.RequireCookie != nil {
		requireCookie = *req.RequireCookie
	}

	preferred := make(map[string]struct{}, len(req.PreferredAccounts))
	for _, accountID := range req.PreferredAccounts {
		accountID = strings.TrimSpace(accountID)
		if accountID != "" {
			preferred[accountID] = struct{}{}
		}
	}

	candidates := make([]SchedulerCandidate, 0, len(accounts))
	for _, account := range accounts {
		if req.TenantID != "" && account.TenantID != req.TenantID {
			continue
		}
		if req.AccountID != "" && account.AccountID != req.AccountID {
			continue
		}
		if len(preferred) > 0 {
			if _, ok := preferred[account.AccountID]; !ok {
				continue
			}
		}

		cookiePresent := fileExists(account.CookiePath)
		available := true
		reasons := make([]string, 0, 4)
		action := "submit_publish_async"

		if requireCookie && !cookiePresent {
			available = false
			reasons = append(reasons, "cookie_missing")
			action = "check_login_status"
		}
		if account.CooldownUntil != "" {
			available = false
			reasons = append(reasons, "account_in_cooldown")
			action = "wait"
		}
		if account.QueueCapacity > 0 && account.QueueDepth >= account.QueueCapacity {
			available = false
			reasons = append(reasons, "account_queue_full")
			action = "wait_or_pick_another_account"
		}
		if account.ActiveJobs >= max(1, account.QueueWorkers) && account.QueueDepth > 0 {
			reasons = append(reasons, "account_busy")
		}
		if account.LastErrorCode != "" {
			reasons = append(reasons, "recent_error:"+account.LastErrorCode)
		}

		score := account.ActiveJobs*100 + account.QueueDepth*10 + account.CurrentInFlight*5
		if !cookiePresent {
			score += 500
		}
		if account.CooldownUntil != "" {
			score += 10000
		}

		candidates = append(candidates, SchedulerCandidate{
			TenantID:          account.TenantID,
			AccountID:         account.AccountID,
			Available:         available,
			Score:             score,
			Reasons:           reasons,
			RecommendedAction: action,
			CookiePresent:     cookiePresent,
			CurrentInFlight:   account.CurrentInFlight,
			MaxConcurrency:    account.MaxConcurrency,
			QueueDepth:        account.QueueDepth,
			QueueCapacity:     account.QueueCapacity,
			ActiveJobs:        account.ActiveJobs,
			QueueWorkers:      account.QueueWorkers,
			CooldownUntil:     account.CooldownUntil,
			LastSuccessAt:     account.LastSuccessAt,
			LastErrorAt:       account.LastErrorAt,
			LastErrorCode:     account.LastErrorCode,
		})
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Available != candidates[j].Available {
			return candidates[i].Available
		}
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score < candidates[j].Score
		}
		if candidates[i].QueueDepth != candidates[j].QueueDepth {
			return candidates[i].QueueDepth < candidates[j].QueueDepth
		}
		if candidates[i].ActiveJobs != candidates[j].ActiveJobs {
			return candidates[i].ActiveJobs < candidates[j].ActiveJobs
		}
		if candidates[i].TenantID == candidates[j].TenantID {
			return candidates[i].AccountID < candidates[j].AccountID
		}
		return candidates[i].TenantID < candidates[j].TenantID
	})

	total := len(candidates)
	availableCount := 0
	for _, candidate := range candidates {
		if candidate.Available {
			availableCount++
		}
	}

	limit := req.Limit
	if limit <= 0 || limit > len(candidates) {
		limit = len(candidates)
	}
	candidates = candidates[:limit]

	var selected *SchedulerCandidate
	if len(candidates) > 0 && candidates[0].Available {
		top := candidates[0]
		selected = &top
	}

	return SchedulerRecommendationResponse{
		Operation:           "publish_scheduler",
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339),
		SelectedAccount:     selected,
		Candidates:          candidates,
		TotalCandidates:     total,
		AvailableCandidates: availableCount,
	}
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

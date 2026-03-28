package main

type SchedulerRecommendationRequest struct {
	AccountScope
	PreferredAccounts []string `json:"preferred_accounts,omitempty"`
	Limit             int      `json:"limit,omitempty"`
	RequireCookie     *bool    `json:"require_cookie,omitempty"`
}

type SchedulerCandidate struct {
	TenantID          string   `json:"tenant_id"`
	AccountID         string   `json:"account_id"`
	Available         bool     `json:"available"`
	Score             int      `json:"score"`
	Reasons           []string `json:"reasons,omitempty"`
	RecommendedAction string   `json:"recommended_action,omitempty"`
	CookiePresent     bool     `json:"cookie_present"`
	CurrentInFlight   int      `json:"current_in_flight"`
	MaxConcurrency    int      `json:"max_concurrency"`
	QueueDepth        int      `json:"queue_depth"`
	QueueCapacity     int      `json:"queue_capacity"`
	ActiveJobs        int      `json:"active_jobs"`
	QueueWorkers      int      `json:"queue_workers"`
	CooldownUntil     string   `json:"cooldown_until,omitempty"`
	LastSuccessAt     string   `json:"last_success_at,omitempty"`
	LastErrorAt       string   `json:"last_error_at,omitempty"`
	LastErrorCode     string   `json:"last_error_code,omitempty"`
}

type SchedulerRecommendationResponse struct {
	Operation           string               `json:"operation"`
	GeneratedAt         string               `json:"generated_at"`
	SelectedAccount     *SchedulerCandidate  `json:"selected_account,omitempty"`
	Candidates          []SchedulerCandidate `json:"candidates"`
	TotalCandidates     int                  `json:"total_candidates"`
	AvailableCandidates int                  `json:"available_candidates"`
}

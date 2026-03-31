package main

type PublishMode string

const (
	PublishModeSync  PublishMode = "sync"
	PublishModeAsync PublishMode = "async"
)

type PublishBackfillStatus string

const (
	PublishBackfillResolved PublishBackfillStatus = "resolved"
	PublishBackfillPending  PublishBackfillStatus = "pending"
	PublishBackfillSkipped  PublishBackfillStatus = "skipped"
)

type PublishCleanupResult struct {
	Status string   `json:"cleanup_status,omitempty"`
	Paths  []string `json:"cleanup_paths,omitempty"`
	Errors []string `json:"cleanup_errors,omitempty"`
}

type ProductBindingResult struct {
	Status            string   `json:"product_bind_status,omitempty"`
	Count             int      `json:"product_bind_count,omitempty"`
	ProductsRequested []string `json:"products_requested,omitempty"`
	ProductsResolved  []string `json:"products_resolved,omitempty"`
	ProductsMissing   []string `json:"products_missing,omitempty"`
	VerifyConfidence  float64  `json:"product_verify_confidence,omitempty"`
}

type PublishVerificationResult struct {
	PublishVisible bool   `json:"publish_visible"`
	ProductVisible *bool  `json:"product_visible,omitempty"`
	CoverVisible   bool   `json:"cover_visible"`
	VerifyStatus   string `json:"verify_status,omitempty"`
	VerifyReason   string `json:"verify_reason,omitempty"`
}

type PublishExecutionResult struct {
	OK             bool   `json:"ok"`
	Status         string `json:"status,omitempty"`
	Mode           string `json:"mode,omitempty"`
	JobID          string `json:"job_id,omitempty"`
	TaskID         string `json:"task_id,omitempty"`
	BatchID        string `json:"batch_id,omitempty"`
	TenantID       string `json:"tenant_id,omitempty"`
	AccountID      string `json:"account_id,omitempty"`
	Title          string `json:"title,omitempty"`
	Content        string `json:"content,omitempty"`
	NoteID         string `json:"note_id,omitempty"`
	NoteURL        string `json:"note_url,omitempty"`
	FeedID         string `json:"feed_id,omitempty"`
	XsecToken      string `json:"xsec_token,omitempty"`
	PublishedAt    string `json:"published_at,omitempty"`
	PublishStartAt string `json:"publish_start_at,omitempty"`
	PublishEndAt   string `json:"publish_end_at,omitempty"`
	DurationMs     int64  `json:"duration_ms,omitempty"`
	BackfillStatus string `json:"backfill_status,omitempty"`
	BackfillReason string `json:"backfill_reason,omitempty"`
	ErrorCode      string `json:"error_code,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`

	ProductBindingResult
	PublishCleanupResult
	PublishVerificationResult
}

type PublishAcceptedResponse struct {
	Accepted  bool   `json:"accepted"`
	JobID     string `json:"job_id"`
	Status    string `json:"status"`
	TaskID    string `json:"task_id,omitempty"`
	BatchID   string `json:"batch_id,omitempty"`
	TenantID  string `json:"tenant_id,omitempty"`
	AccountID string `json:"account_id,omitempty"`
	Mode      string `json:"mode,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

type PublishJobStatusResponse struct {
	JobID      string                  `json:"job_id"`
	Status     string                  `json:"status"`
	TaskID     string                  `json:"task_id,omitempty"`
	BatchID    string                  `json:"batch_id,omitempty"`
	TenantID   string                  `json:"tenant_id,omitempty"`
	AccountID  string                  `json:"account_id,omitempty"`
	Mode       string                  `json:"mode,omitempty"`
	CreatedAt  string                  `json:"created_at,omitempty"`
	StartedAt  string                  `json:"started_at,omitempty"`
	FinishedAt string                  `json:"finished_at,omitempty"`
	Result     *PublishExecutionResult `json:"result,omitempty"`
}

type PublishJobStatusRequest struct {
	JobID string `json:"job_id" binding:"required"`
}

type RecentPublishedNotesRequest struct {
	AccountScope
	SinceTime    string `json:"since_time,omitempty"`
	TitleKeyword string `json:"title_keyword,omitempty"`
	Limit        int    `json:"limit,omitempty"`
}

type RecentPublishedNote struct {
	NoteID      string `json:"note_id,omitempty"`
	NoteURL     string `json:"note_url,omitempty"`
	FeedID      string `json:"feed_id,omitempty"`
	XsecToken   string `json:"xsec_token,omitempty"`
	Title       string `json:"title,omitempty"`
	PublishTime string `json:"publish_time,omitempty"`
}

type RecentPublishedNotesResponse struct {
	Notes []RecentPublishedNote `json:"notes"`
	Count int                   `json:"count"`
}

type VerifyPublishedNoteRequest struct {
	AccountScope
	JobID     string `json:"job_id,omitempty"`
	NoteID    string `json:"note_id,omitempty"`
	FeedID    string `json:"feed_id,omitempty"`
	XsecToken string `json:"xsec_token,omitempty"`
}

type VerifyPublishedNoteResponse struct {
	JobID       string `json:"job_id,omitempty"`
	NoteID      string `json:"note_id,omitempty"`
	NoteURL     string `json:"note_url,omitempty"`
	FeedID      string `json:"feed_id,omitempty"`
	XsecToken   string `json:"xsec_token,omitempty"`
	AccountID   string `json:"account_id,omitempty"`
	PublishedAt string `json:"published_at,omitempty"`

	PublishVerificationResult
	ProductBindingResult
}

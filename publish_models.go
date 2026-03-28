package main

type PublishExecutionResult struct {
	OK               bool     `json:"ok"`
	Mode             string   `json:"mode,omitempty"`
	Operation        string   `json:"operation,omitempty"`
	TaskID           string   `json:"task_id,omitempty"`
	BatchID          string   `json:"batch_id,omitempty"`
	TenantID         string   `json:"tenant_id,omitempty"`
	AccountID        string   `json:"account_id,omitempty"`
	Title            string   `json:"title,omitempty"`
	Content          string   `json:"content,omitempty"`
	Status           string   `json:"status,omitempty"`
	NoteID           string   `json:"note_id,omitempty"`
	NoteURL          string   `json:"note_url,omitempty"`
	FeedID           string   `json:"feed_id,omitempty"`
	XsecToken        string   `json:"xsec_token,omitempty"`
	DurationMs       int64    `json:"duration_ms"`
	ImagePath        string   `json:"image_path,omitempty"`
	ImagePaths       []string `json:"image_paths,omitempty"`
	StagedImagePaths []string `json:"staged_image_paths,omitempty"`
	VideoPath        string   `json:"video_path,omitempty"`
	PublishStartAt   string   `json:"publish_start_at,omitempty"`
	PublishEndAt     string   `json:"publish_end_at,omitempty"`
	ErrorCode        string   `json:"error_code,omitempty"`
	ErrorMessage     string   `json:"error_message,omitempty"`
	Retryable        bool     `json:"retryable,omitempty"`
}

type PublishMode string

const (
	PublishModeSync  PublishMode = "sync"
	PublishModeAsync PublishMode = "async"
)

type PublishJobStatus string

const (
	PublishJobStatusAccepted  PublishJobStatus = "accepted"
	PublishJobStatusRunning   PublishJobStatus = "running"
	PublishJobStatusSucceeded PublishJobStatus = "succeeded"
	PublishJobStatusFailed    PublishJobStatus = "failed"
	PublishJobStatusCanceled  PublishJobStatus = "canceled"
)

type PublishAcceptedResponse struct {
	Accepted      bool             `json:"accepted"`
	JobID         string           `json:"job_id"`
	Status        PublishJobStatus `json:"status"`
	Mode          string           `json:"mode"`
	Operation     string           `json:"operation"`
	TaskID        string           `json:"task_id,omitempty"`
	BatchID       string           `json:"batch_id,omitempty"`
	TenantID      string           `json:"tenant_id,omitempty"`
	AccountID     string           `json:"account_id,omitempty"`
	CreatedAt     string           `json:"created_at,omitempty"`
	QueueDepth    int              `json:"queue_depth,omitempty"`
	QueueCapacity int              `json:"queue_capacity,omitempty"`
	ActiveJobs    int              `json:"active_jobs,omitempty"`
}

type PublishJobStatusResponse struct {
	Accepted      bool                    `json:"accepted"`
	JobID         string                  `json:"job_id"`
	Status        PublishJobStatus        `json:"status"`
	Mode          string                  `json:"mode"`
	Operation     string                  `json:"operation"`
	TaskID        string                  `json:"task_id,omitempty"`
	BatchID       string                  `json:"batch_id,omitempty"`
	TenantID      string                  `json:"tenant_id,omitempty"`
	AccountID     string                  `json:"account_id,omitempty"`
	CreatedAt     string                  `json:"created_at,omitempty"`
	StartedAt     string                  `json:"started_at,omitempty"`
	FinishedAt    string                  `json:"finished_at,omitempty"`
	QueueDepth    int                     `json:"queue_depth,omitempty"`
	QueueCapacity int                     `json:"queue_capacity,omitempty"`
	ActiveJobs    int                     `json:"active_jobs,omitempty"`
	Result        *PublishExecutionResult `json:"result,omitempty"`
}

type StageImagesRequest struct {
	Images []string `json:"images" binding:"required,min=1"`
}

type StageImagesResponse struct {
	OK               bool     `json:"ok"`
	ImagePaths       []string `json:"image_paths"`
	StagedImagePaths []string `json:"staged_image_paths"`
	Count            int      `json:"count"`
}

type PublishJobStatusRequest struct {
	JobID string `json:"job_id" binding:"required"`
}

package main

import "github.com/hekaixin66-sketch/xiaohongshuritter/xiaohongshu"

// ErrorResponse HTTP error payload.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details any    `json:"details,omitempty"`
}

// SuccessResponse HTTP success payload.
type SuccessResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data"`
	Message string `json:"message,omitempty"`
}

// MCPToolResult internal MCP conversion payload.
type MCPToolResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// MCPContent internal MCP content model.
type MCPContent struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// CommentLoadConfig comment crawling config.
type CommentLoadConfig struct {
	ClickMoreReplies    bool   `json:"click_more_replies,omitempty"`
	MaxRepliesThreshold int    `json:"max_replies_threshold,omitempty"`
	MaxCommentItems     int    `json:"max_comment_items,omitempty"`
	ScrollSpeed         string `json:"scroll_speed,omitempty"`
}

// FeedDetailRequest feed detail request.
type FeedDetailRequest struct {
	AccountScope
	FeedID          string             `json:"feed_id" binding:"required"`
	XsecToken       string             `json:"xsec_token" binding:"required"`
	LoadAllComments bool               `json:"load_all_comments,omitempty"`
	CommentConfig   *CommentLoadConfig `json:"comment_config,omitempty"`
}

type SearchFeedsRequest struct {
	AccountScope
	Keyword string                   `json:"keyword" binding:"required"`
	Filters xiaohongshu.FilterOption `json:"filters,omitempty"`
}

// FeedDetailResponse feed detail response.
type FeedDetailResponse struct {
	FeedID string `json:"feed_id"`
	Data   any    `json:"data"`
}

// PostCommentRequest comment request.
type PostCommentRequest struct {
	AccountScope
	FeedID    string `json:"feed_id" binding:"required"`
	XsecToken string `json:"xsec_token" binding:"required"`
	Content   string `json:"content" binding:"required"`
}

// PostCommentResponse comment response.
type PostCommentResponse struct {
	FeedID  string `json:"feed_id"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ReplyCommentRequest reply comment request.
type ReplyCommentRequest struct {
	AccountScope
	FeedID    string `json:"feed_id" binding:"required"`
	XsecToken string `json:"xsec_token" binding:"required"`
	CommentID string `json:"comment_id" binding:"required_without=UserID"`
	UserID    string `json:"user_id" binding:"required_without=CommentID"`
	Content   string `json:"content" binding:"required"`
}

// ReplyCommentResponse reply response.
type ReplyCommentResponse struct {
	FeedID          string `json:"feed_id"`
	TargetCommentID string `json:"target_comment_id,omitempty"`
	TargetUserID    string `json:"target_user_id,omitempty"`
	Success         bool   `json:"success"`
	Message         string `json:"message"`
}

// UserProfileRequest user profile request.
type UserProfileRequest struct {
	AccountScope
	UserID    string `json:"user_id" binding:"required"`
	XsecToken string `json:"xsec_token" binding:"required"`
}

// ActionResult general action response.
type ActionResult struct {
	FeedID  string `json:"feed_id"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

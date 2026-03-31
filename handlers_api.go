package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hekaixin66-sketch/xiaohongshuritter/xiaohongshu"
	"github.com/sirupsen/logrus"
)

func respondError(c *gin.Context, statusCode int, code, message string, details any) {
	response := ErrorResponse{
		Error:   message,
		Code:    code,
		Details: details,
	}

	logrus.Errorf("%s %s %s %d", c.Request.Method, c.Request.URL.Path,
		c.GetString("account"), statusCode)

	c.JSON(statusCode, response)
}

func respondSuccess(c *gin.Context, data any, message string) {
	response := SuccessResponse{
		Success: true,
		Data:    data,
		Message: message,
	}

	logrus.Infof("%s %s %s %d", c.Request.Method, c.Request.URL.Path,
		c.GetString("account"), http.StatusOK)

	c.JSON(http.StatusOK, response)
}

func (s *AppServer) checkLoginStatusHandler(c *gin.Context) {
	ctx, scope, err := s.resolveScopeForHTTP(c, AccountScope{})
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ACCOUNT_SCOPE", "invalid tenant/account", err.Error())
		return
	}

	status, err := s.xiaohongshuService.CheckLoginStatus(ctx)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "STATUS_CHECK_FAILED", "failed to check login status", err.Error())
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, status, "ok")
}

func (s *AppServer) getLoginQrcodeHandler(c *gin.Context) {
	ctx, scope, err := s.resolveScopeForHTTP(c, AccountScope{})
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ACCOUNT_SCOPE", "invalid tenant/account", err.Error())
		return
	}

	result, err := s.xiaohongshuService.GetLoginQrcode(ctx)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "GET_LOGIN_QRCODE_FAILED", "failed to get login qrcode", err.Error())
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, result, "ok")
}

func (s *AppServer) deleteCookiesHandler(c *gin.Context) {
	ctx, scope, err := s.resolveScopeForHTTP(c, AccountScope{})
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ACCOUNT_SCOPE", "invalid tenant/account", err.Error())
		return
	}

	cookiePath, err := s.xiaohongshuService.DeleteCookies(ctx)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "DELETE_COOKIES_FAILED", "failed to delete cookies", err.Error())
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, map[string]any{
		"cookie_path": cookiePath,
		"scope":       scopeLabel(scope),
	}, "ok")
}

func (s *AppServer) publishAsyncHandler(c *gin.Context) {
	var req PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", err.Error())
		return
	}
	req.Mode = string(PublishModeAsync)

	ctx, scope, err := s.resolveScopeForHTTP(c, req.AccountScope)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ACCOUNT_SCOPE", "invalid tenant/account", err.Error())
		return
	}
	_ = ctx

	result, err := s.jobManager.SubmitContent(scope, &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "PUBLISH_ASYNC_FAILED", "publish async failed", err.Error())
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, result, "accepted")
}

func (s *AppServer) publishVideoAsyncHandler(c *gin.Context) {
	var req PublishVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", err.Error())
		return
	}
	req.Mode = string(PublishModeAsync)

	ctx, scope, err := s.resolveScopeForHTTP(c, req.AccountScope)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ACCOUNT_SCOPE", "invalid tenant/account", err.Error())
		return
	}
	_ = ctx

	result, err := s.jobManager.SubmitVideo(scope, &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "PUBLISH_VIDEO_ASYNC_FAILED", "publish video async failed", err.Error())
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, result, "accepted")
}

func (s *AppServer) publishJobStatusHandler(c *gin.Context) {
	jobID := strings.TrimSpace(c.Param("job_id"))
	if jobID == "" {
		respondError(c, http.StatusBadRequest, "MISSING_JOB_ID", "job_id is required", nil)
		return
	}

	result, err := s.jobManager.Get(jobID)
	if err != nil {
		respondError(c, http.StatusNotFound, "JOB_NOT_FOUND", "publish job not found", err.Error())
		return
	}

	c.Set("account", "system")
	respondSuccess(c, result, "ok")
}

func (s *AppServer) verifyPublishedNoteHandler(c *gin.Context) {
	var req VerifyPublishedNoteRequest
	switch c.Request.Method {
	case http.MethodPost:
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", err.Error())
			return
		}
	default:
		req.JobID = c.Query("job_id")
		req.NoteID = c.Query("note_id")
		req.FeedID = c.Query("feed_id")
		req.XsecToken = c.Query("xsec_token")
	}

	ctx, scope, err := s.resolveScopeForHTTP(c, req.AccountScope)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ACCOUNT_SCOPE", "invalid tenant/account", err.Error())
		return
	}

	result, err := s.VerifyPublishedNote(ctx, &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "VERIFY_PUBLISHED_NOTE_FAILED", "failed to verify published note", err.Error())
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, result, "ok")
}

func (s *AppServer) publishHandler(c *gin.Context) {
	var req PublishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", err.Error())
		return
	}

	ctx, scope, err := s.resolveScopeForHTTP(c, req.AccountScope)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ACCOUNT_SCOPE", "invalid tenant/account", err.Error())
		return
	}

	if strings.EqualFold(strings.TrimSpace(req.Mode), string(PublishModeAsync)) {
		result, submitErr := s.jobManager.SubmitContent(scope, &req)
		if submitErr != nil {
			respondError(c, http.StatusInternalServerError, "PUBLISH_ASYNC_FAILED", "publish async failed", submitErr.Error())
			return
		}
		c.Set("account", scopeLabel(scope))
		respondSuccess(c, result, "accepted")
		return
	}

	result, err := s.xiaohongshuService.PublishContent(ctx, &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "PUBLISH_FAILED", "publish failed", result)
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, result, "ok")
}

func (s *AppServer) publishVideoHandler(c *gin.Context) {
	var req PublishVideoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", err.Error())
		return
	}

	ctx, scope, err := s.resolveScopeForHTTP(c, req.AccountScope)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ACCOUNT_SCOPE", "invalid tenant/account", err.Error())
		return
	}

	if strings.EqualFold(strings.TrimSpace(req.Mode), string(PublishModeAsync)) {
		result, submitErr := s.jobManager.SubmitVideo(scope, &req)
		if submitErr != nil {
			respondError(c, http.StatusInternalServerError, "PUBLISH_VIDEO_ASYNC_FAILED", "publish video async failed", submitErr.Error())
			return
		}
		c.Set("account", scopeLabel(scope))
		respondSuccess(c, result, "accepted")
		return
	}

	result, err := s.xiaohongshuService.PublishVideo(ctx, &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "PUBLISH_VIDEO_FAILED", "publish video failed", result)
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, result, "ok")
}

func (s *AppServer) recentPublishedNotesHandler(c *gin.Context) {
	var req RecentPublishedNotesRequest
	switch c.Request.Method {
	case http.MethodPost:
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", err.Error())
			return
		}
	default:
		req.TitleKeyword = c.Query("title_keyword")
		req.SinceTime = c.Query("since_time")
		if limit := strings.TrimSpace(c.Query("limit")); limit != "" {
			if parsed, err := strconv.Atoi(limit); err == nil {
				req.Limit = parsed
			}
		}
	}

	ctx, scope, err := s.resolveScopeForHTTP(c, req.AccountScope)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ACCOUNT_SCOPE", "invalid tenant/account", err.Error())
		return
	}

	result, err := s.xiaohongshuService.ListRecentPublishedNotes(ctx, &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "LIST_RECENT_NOTES_FAILED", "failed to list recent published notes", err.Error())
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, result, "ok")
}

func (s *AppServer) listFeedsHandler(c *gin.Context) {
	ctx, scope, err := s.resolveScopeForHTTP(c, AccountScope{})
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ACCOUNT_SCOPE", "invalid tenant/account", err.Error())
		return
	}

	result, err := s.xiaohongshuService.ListFeeds(ctx)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "LIST_FEEDS_FAILED", "failed to list feeds", err.Error())
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, result, "ok")
}

func (s *AppServer) searchFeedsHandler(c *gin.Context) {
	var (
		keyword string
		filters xiaohongshu.FilterOption
		body    AccountScope
	)

	switch c.Request.Method {
	case http.MethodPost:
		var req SearchFeedsRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", err.Error())
			return
		}
		keyword = req.Keyword
		filters = req.Filters
		body = req.AccountScope
	default:
		keyword = c.Query("keyword")
	}

	if keyword == "" {
		respondError(c, http.StatusBadRequest, "MISSING_KEYWORD", "keyword is required", "keyword")
		return
	}

	ctx, scope, err := s.resolveScopeForHTTP(c, body)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ACCOUNT_SCOPE", "invalid tenant/account", err.Error())
		return
	}

	result, err := s.xiaohongshuService.SearchFeeds(ctx, keyword, filters)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "SEARCH_FEEDS_FAILED", "failed to search feeds", err.Error())
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, result, "ok")
}

func (s *AppServer) getFeedDetailHandler(c *gin.Context) {
	var req FeedDetailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", err.Error())
		return
	}

	ctx, scope, err := s.resolveScopeForHTTP(c, req.AccountScope)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ACCOUNT_SCOPE", "invalid tenant/account", err.Error())
		return
	}

	var result *FeedDetailResponse
	if req.CommentConfig != nil {
		cfg := xiaohongshu.CommentLoadConfig{
			ClickMoreReplies:    req.CommentConfig.ClickMoreReplies,
			MaxRepliesThreshold: req.CommentConfig.MaxRepliesThreshold,
			MaxCommentItems:     req.CommentConfig.MaxCommentItems,
			ScrollSpeed:         req.CommentConfig.ScrollSpeed,
		}
		result, err = s.xiaohongshuService.GetFeedDetailWithConfig(ctx, req.FeedID, req.XsecToken, req.LoadAllComments, cfg)
	} else {
		result, err = s.xiaohongshuService.GetFeedDetail(ctx, req.FeedID, req.XsecToken, req.LoadAllComments)
	}

	if err != nil {
		respondError(c, http.StatusInternalServerError, "GET_FEED_DETAIL_FAILED", "failed to get feed detail", err.Error())
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, result, "ok")
}

func (s *AppServer) userProfileHandler(c *gin.Context) {
	var req UserProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", err.Error())
		return
	}

	ctx, scope, err := s.resolveScopeForHTTP(c, req.AccountScope)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ACCOUNT_SCOPE", "invalid tenant/account", err.Error())
		return
	}

	result, err := s.xiaohongshuService.UserProfile(ctx, req.UserID, req.XsecToken)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "GET_USER_PROFILE_FAILED", "failed to get user profile", err.Error())
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, map[string]any{"data": result}, "ok")
}

func (s *AppServer) postCommentHandler(c *gin.Context) {
	var req PostCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", err.Error())
		return
	}

	ctx, scope, err := s.resolveScopeForHTTP(c, req.AccountScope)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ACCOUNT_SCOPE", "invalid tenant/account", err.Error())
		return
	}

	result, err := s.xiaohongshuService.PostCommentToFeed(ctx, req.FeedID, req.XsecToken, req.Content)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "POST_COMMENT_FAILED", "failed to post comment", err.Error())
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, result, result.Message)
}

func (s *AppServer) replyCommentHandler(c *gin.Context) {
	var req ReplyCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", err.Error())
		return
	}

	ctx, scope, err := s.resolveScopeForHTTP(c, req.AccountScope)
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ACCOUNT_SCOPE", "invalid tenant/account", err.Error())
		return
	}

	result, err := s.xiaohongshuService.ReplyCommentToFeed(ctx, req.FeedID, req.XsecToken, req.CommentID, req.UserID, req.Content)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "REPLY_COMMENT_FAILED", "failed to reply comment", err.Error())
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, result, result.Message)
}

func healthHandler(c *gin.Context) {
	c.Set("account", "system")
	respondSuccess(c, map[string]any{
		"status":    "healthy",
		"service":   "xiaohongshuritter",
		"timestamp": "now",
	}, "ok")
}

func (s *AppServer) myProfileHandler(c *gin.Context) {
	ctx, scope, err := s.resolveScopeForHTTP(c, AccountScope{})
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ACCOUNT_SCOPE", "invalid tenant/account", err.Error())
		return
	}

	result, err := s.xiaohongshuService.GetMyProfile(ctx)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "GET_MY_PROFILE_FAILED", "failed to get my profile", err.Error())
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, map[string]any{"data": result}, "ok")
}

func (s *AppServer) listAccountsHandler(c *gin.Context) {
	c.Set("account", "system")
	accounts := s.xiaohongshuService.ListAccounts()
	respondSuccess(c, map[string]any{
		"config_path": s.xiaohongshuService.AccountConfigPath(),
		"accounts":    accounts,
		"count":       len(accounts),
	}, "ok")
}

package main

import (
	"net/http"

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

func respondAppError(c *gin.Context, err error, details any) {
	appErr := classifyError(err)
	respondError(c, appErr.StatusCode, appErr.Code, appErr.Message, mergeErrorDetails(err, details))
}

func (s *AppServer) checkLoginStatusHandler(c *gin.Context) {
	ctx, scope, err := s.resolveScopeForHTTP(c, AccountScope{})
	if err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_ACCOUNT_SCOPE", "invalid tenant/account", err.Error())
		return
	}
	ctx, cancel := s.withOperationTimeout(ctx, OperationCheckLoginStatus)
	defer cancel()

	status, err := s.xiaohongshuService.CheckLoginStatus(ctx)
	if err != nil {
		respondAppError(c, err, nil)
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
	ctx, cancel := s.withOperationTimeout(ctx, OperationGetLoginQRCode)
	defer cancel()

	result, err := s.xiaohongshuService.GetLoginQrcode(ctx)
	if err != nil {
		respondAppError(c, err, nil)
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
	ctx, cancel := s.withOperationTimeout(ctx, OperationDeleteCookies)
	defer cancel()

	cookiePath, err := s.xiaohongshuService.DeleteCookies(ctx)
	if err != nil {
		respondAppError(c, err, nil)
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, map[string]any{
		"cookie_path": cookiePath,
		"scope":       scopeLabel(scope),
	}, "ok")
}

func (s *AppServer) stageImagesHandler(c *gin.Context) {
	var req StageImagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", err.Error())
		return
	}

	_, cancel := s.withOperationTimeout(c.Request.Context(), OperationStageImagePublish)
	defer cancel()

	result, err := s.xiaohongshuService.StageImages(req.Images)
	if err != nil {
		respondAppError(c, err, nil)
		return
	}

	c.Set("account", "system")
	respondSuccess(c, result, "ok")
}

func (s *AppServer) publishJobStatusHandler(c *gin.Context) {
	jobID := c.Param("job_id")
	if jobID == "" {
		respondError(c, http.StatusBadRequest, "JOB_ID_REQUIRED", "job_id is required", nil)
		return
	}

	_, cancel := s.withOperationTimeout(c.Request.Context(), OperationPublishJobStatus)
	defer cancel()

	result, err := s.jobManager.Get(jobID)
	if err != nil {
		respondAppError(c, err, nil)
		return
	}

	c.Set("account", "system")
	respondSuccess(c, result, "ok")
}

func (s *AppServer) recommendAccountsHandler(c *gin.Context) {
	var req SchedulerRecommendationRequest
	if c.Request.Method == http.MethodPost {
		if err := c.ShouldBindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid request", err.Error())
			return
		}
	} else {
		req.TenantID = c.Query("tenant_id")
		req.AccountID = c.Query("account_id")
	}

	_, cancel := s.withOperationTimeout(c.Request.Context(), OperationListAccounts)
	defer cancel()

	result := s.RecommendAccountsForPublish(req)
	c.Set("account", "system")
	respondSuccess(c, result, "ok")
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
	ctx = withOperationMetadata(ctx, operationMetadata{Name: OperationPublishContentAsync, TaskID: req.TaskID, BatchID: req.BatchID})
	_, cancel := s.withOperationTimeout(ctx, OperationPublishContentAsync)
	defer cancel()

	result, err := s.jobManager.SubmitContent(scope, &req)
	if err != nil {
		respondAppError(c, err, nil)
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
	ctx = withOperationMetadata(ctx, operationMetadata{Name: OperationPublishVideoAsync, TaskID: req.TaskID, BatchID: req.BatchID})
	_, cancel := s.withOperationTimeout(ctx, OperationPublishVideoAsync)
	defer cancel()

	result, err := s.jobManager.SubmitVideo(scope, &req)
	if err != nil {
		respondAppError(c, err, nil)
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, result, "accepted")
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
	op := OperationPublishContent
	if isAsyncMode(req.Mode) {
		op = OperationPublishContentAsync
	}
	ctx = withOperationMetadata(ctx, operationMetadata{Name: op, TaskID: req.TaskID, BatchID: req.BatchID})
	ctx, cancel := s.withOperationTimeout(ctx, op)
	defer cancel()

	if isAsyncMode(req.Mode) {
		result, submitErr := s.jobManager.SubmitContent(scope, &req)
		if submitErr != nil {
			respondAppError(c, submitErr, nil)
			return
		}
		c.Set("account", scopeLabel(scope))
		respondSuccess(c, result, "accepted")
		return
	}

	result, err := s.xiaohongshuService.PublishContent(ctx, &req)
	if err != nil {
		respondAppError(c, err, result)
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
	op := OperationPublishVideo
	if isAsyncMode(req.Mode) {
		op = OperationPublishVideoAsync
	}
	ctx = withOperationMetadata(ctx, operationMetadata{Name: op, TaskID: req.TaskID, BatchID: req.BatchID})
	ctx, cancel := s.withOperationTimeout(ctx, op)
	defer cancel()

	if isAsyncMode(req.Mode) {
		result, submitErr := s.jobManager.SubmitVideo(scope, &req)
		if submitErr != nil {
			respondAppError(c, submitErr, nil)
			return
		}
		c.Set("account", scopeLabel(scope))
		respondSuccess(c, result, "accepted")
		return
	}

	result, err := s.xiaohongshuService.PublishVideo(ctx, &req)
	if err != nil {
		respondAppError(c, err, result)
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
	ctx, cancel := s.withOperationTimeout(ctx, OperationListFeeds)
	defer cancel()

	result, err := s.xiaohongshuService.ListFeeds(ctx)
	if err != nil {
		respondAppError(c, err, nil)
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
	ctx, cancel := s.withOperationTimeout(ctx, OperationSearchFeeds)
	defer cancel()

	result, err := s.xiaohongshuService.SearchFeeds(ctx, keyword, filters)
	if err != nil {
		respondAppError(c, err, nil)
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
	ctx, cancel := s.withOperationTimeout(ctx, OperationGetFeedDetail)
	defer cancel()

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
		respondAppError(c, err, nil)
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
	ctx, cancel := s.withOperationTimeout(ctx, OperationUserProfile)
	defer cancel()

	result, err := s.xiaohongshuService.UserProfile(ctx, req.UserID, req.XsecToken)
	if err != nil {
		respondAppError(c, err, nil)
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
	ctx, cancel := s.withOperationTimeout(ctx, OperationPostComment)
	defer cancel()

	result, err := s.xiaohongshuService.PostCommentToFeed(ctx, req.FeedID, req.XsecToken, req.Content)
	if err != nil {
		respondAppError(c, err, nil)
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
	ctx, cancel := s.withOperationTimeout(ctx, OperationReplyComment)
	defer cancel()

	result, err := s.xiaohongshuService.ReplyCommentToFeed(ctx, req.FeedID, req.XsecToken, req.CommentID, req.UserID, req.Content)
	if err != nil {
		respondAppError(c, err, nil)
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
	ctx, cancel := s.withOperationTimeout(ctx, OperationUserProfile)
	defer cancel()

	result, err := s.xiaohongshuService.GetMyProfile(ctx)
	if err != nil {
		respondAppError(c, err, nil)
		return
	}

	c.Set("account", scopeLabel(scope))
	respondSuccess(c, map[string]any{"data": result}, "ok")
}

func (s *AppServer) listAccountsHandler(c *gin.Context) {
	c.Set("account", "system")
	ctx, cancel := s.withOperationTimeout(c.Request.Context(), OperationListAccounts)
	defer cancel()
	_ = ctx
	queueStats := s.jobManager.QueueStats()
	accounts := decorateAccountInfosWithQueueStats(s.xiaohongshuService.ListAccounts(), queueStats)
	respondSuccess(c, map[string]any{
		"config_path":      s.xiaohongshuService.AccountConfigPath(),
		"accounts":         accounts,
		"count":            len(accounts),
		"runtime":          s.xiaohongshuService.RuntimeStats(),
		"job_runtime":      s.jobManager.RuntimeStats(),
		"job_queues":       queueStats,
		"global_in_flight": s.xiaohongshuService.accountManager.GlobalInFlight(),
	}, "ok")
}

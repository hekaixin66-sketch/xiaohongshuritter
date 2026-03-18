package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/hekaixin66-sketch/xiaohongshuritter/xiaohongshu"
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

	result, err := s.xiaohongshuService.PublishContent(ctx, &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "PUBLISH_FAILED", "publish failed", err.Error())
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

	result, err := s.xiaohongshuService.PublishVideo(ctx, &req)
	if err != nil {
		respondError(c, http.StatusInternalServerError, "PUBLISH_VIDEO_FAILED", "publish video failed", err.Error())
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

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/hekaixin66-sketch/xiaohongshuritter/browser"
	"github.com/hekaixin66-sketch/xiaohongshuritter/cookies"
	"github.com/hekaixin66-sketch/xiaohongshuritter/pkg/downloader"
	"github.com/hekaixin66-sketch/xiaohongshuritter/pkg/xhsutil"
	"github.com/hekaixin66-sketch/xiaohongshuritter/xiaohongshu"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/headless_browser"
)

// XiaohongshuService encapsulates business operations.
type XiaohongshuService struct {
	accountManager *AccountManager
	metrics        *ServiceMetrics
	browserGuard   *BrowserGuard
}

func NewXiaohongshuService() (*XiaohongshuService, error) {
	manager, err := NewAccountManagerFromEnv()
	if err != nil {
		return nil, err
	}
	metrics := NewServiceMetrics()
	return &XiaohongshuService{
		accountManager: manager,
		metrics:        metrics,
		browserGuard:   NewBrowserGuard(manager.GlobalConcurrencyLimit(), metrics),
	}, nil
}

func (s *XiaohongshuService) ResolveScope(scope AccountScope) (AccountScope, error) {
	resolved, err := s.accountManager.Resolve(scope)
	if err != nil {
		return AccountScope{}, err
	}
	return AccountScope{TenantID: resolved.TenantID, AccountID: resolved.AccountID}, nil
}

func (s *XiaohongshuService) ListAccounts() []AccountInfo {
	return s.accountManager.ListAccounts()
}

func (s *XiaohongshuService) AccountConfigPath() string {
	return s.accountManager.ConfigPath()
}

func (s *XiaohongshuService) RuntimeStats() RuntimeStats {
	return s.metrics.Snapshot()
}

func (s *XiaohongshuService) StartMaintenance(ctx context.Context) {
	if s == nil || s.browserGuard == nil {
		return
	}
	s.browserGuard.Start(ctx, s.metrics.ActiveBrowserSessions)
}

// PublishRequest is request payload for image posts.
type PublishRequest struct {
	AccountScope
	Title      string   `json:"title" binding:"required"`
	Content    string   `json:"content" binding:"required"`
	Images     []string `json:"images" binding:"required,min=1"`
	Tags       []string `json:"tags,omitempty"`
	ScheduleAt string   `json:"schedule_at,omitempty"`
	IsOriginal bool     `json:"is_original,omitempty"`
	Visibility string   `json:"visibility,omitempty"`
	Products   []string `json:"products,omitempty"`
	TaskID     string   `json:"task_id,omitempty"`
	BatchID    string   `json:"batch_id,omitempty"`
	Mode       string   `json:"mode,omitempty"`
}

// LoginStatusResponse login status payload.
type LoginStatusResponse struct {
	IsLoggedIn bool   `json:"is_logged_in"`
	Username   string `json:"username,omitempty"`
}

// LoginQrcodeResponse login QR payload.
type LoginQrcodeResponse struct {
	Timeout    string `json:"timeout"`
	IsLoggedIn bool   `json:"is_logged_in"`
	Img        string `json:"img,omitempty"`
}

// PublishResponse publish response payload.
type PublishResponse struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Images  int    `json:"images"`
	Status  string `json:"status"`
	PostID  string `json:"post_id,omitempty"`
}

// PublishVideoRequest is request payload for video posts.
type PublishVideoRequest struct {
	AccountScope
	Title      string   `json:"title" binding:"required"`
	Content    string   `json:"content" binding:"required"`
	Video      string   `json:"video" binding:"required"`
	Tags       []string `json:"tags,omitempty"`
	ScheduleAt string   `json:"schedule_at,omitempty"`
	Visibility string   `json:"visibility,omitempty"`
	Products   []string `json:"products,omitempty"`
	TaskID     string   `json:"task_id,omitempty"`
	BatchID    string   `json:"batch_id,omitempty"`
	Mode       string   `json:"mode,omitempty"`
}

// PublishVideoResponse publish response payload.
type PublishVideoResponse struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Video   string `json:"video"`
	Status  string `json:"status"`
	PostID  string `json:"post_id,omitempty"`
}

// FeedsListResponse feeds payload.
type FeedsListResponse struct {
	Feeds []xiaohongshu.Feed `json:"feeds"`
	Count int                `json:"count"`
}

// UserProfileResponse user profile payload.
type UserProfileResponse struct {
	UserBasicInfo xiaohongshu.UserBasicInfo      `json:"userBasicInfo"`
	Interactions  []xiaohongshu.UserInteractions `json:"interactions"`
	Feeds         []xiaohongshu.Feed             `json:"feeds"`
}

func (s *XiaohongshuService) DeleteCookies(ctx context.Context) (string, error) {
	session, err := s.acquireSession(ctx)
	if err != nil {
		return "", err
	}
	defer session.Release()

	cookieLoader := cookies.NewLoadCookie(session.CookiePath)
	if err := cookieLoader.DeleteCookies(); err != nil {
		return "", err
	}
	return session.CookiePath, nil
}

func (s *XiaohongshuService) CheckLoginStatus(ctx context.Context) (*LoginStatusResponse, error) {
	var (
		loggedIn bool
		label    string
	)

	err := s.withAccountPage(ctx, func(page *rod.Page, session *AccountSession) error {
		loginAction := xiaohongshu.NewLogin(page)
		status, err := loginAction.CheckLoginStatus(ctx)
		if err != nil {
			return err
		}
		loggedIn = status
		label = accountLabelFromSession(session)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &LoginStatusResponse{IsLoggedIn: loggedIn, Username: label}, nil
}

func (s *XiaohongshuService) GetLoginQrcode(ctx context.Context) (*LoginQrcodeResponse, error) {
	session, err := s.acquireSession(ctx)
	if err != nil {
		return nil, err
	}
	releaseBrowserSlot, err := s.browserGuard.Acquire(ctx)
	if err != nil {
		session.Release()
		return nil, err
	}

	b := newBrowserForSession(session)
	s.metrics.BrowserSessionStarted()
	page := b.NewPage()
	var cleanupOnce sync.Once

	cleanup := func() {
		cleanupOnce.Do(func() {
			_ = page.Close()
			b.Close()
			session.Release()
			releaseBrowserSlot()
			s.metrics.BrowserSessionEnded()
		})
	}

	loginAction := xiaohongshu.NewLogin(page)
	img, loggedIn, err := loginAction.FetchQrcodeImage(ctx)
	if err != nil || loggedIn {
		cleanup()
	}
	if err != nil {
		return nil, err
	}

	timeout := 4 * time.Minute
	if !loggedIn {
		cookiePath := session.CookiePath
		go func() {
			ctxTimeout, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			defer cleanup()

			if loginAction.WaitForLogin(ctxTimeout) {
				if saveErr := saveCookiesToPath(page, cookiePath); saveErr != nil {
					logrus.WithError(saveErr).WithField("cookie_path", cookiePath).Error("failed to save cookies")
				}
			}
		}()
	}

	return &LoginQrcodeResponse{
		Timeout: func() string {
			if loggedIn {
				return "0s"
			}
			return timeout.String()
		}(),
		Img:        img,
		IsLoggedIn: loggedIn,
	}, nil
}

func (s *XiaohongshuService) PublishContent(ctx context.Context, req *PublishRequest) (result *PublishExecutionResult, err error) {
	scope := AccountScopeFromContext(ctx)
	meta := operationMetadataFromContext(ctx)
	startedAt := time.Now().UTC()

	result = newPublishExecutionResult(OperationPublishContent, scope, req.Title, req.Content, firstNonEmpty(req.TaskID, meta.TaskID), firstNonEmpty(req.BatchID, meta.BatchID))
	result.ImagePaths = copyStrings(req.Images)
	result.PublishStartAt = formatTime(startedAt)

	defer func() {
		s.finishPublishExecution(scope, startedAt, result, err)
	}()

	if len(req.Images) == 0 {
		err = newAppError("IMAGES_REQUIRED", "images are required", 400, false, nil, nil)
		return result, err
	}
	if xhsutil.CalcTitleLength(req.Title) > 20 {
		err = newAppError("TITLE_TOO_LONG", "title length exceeds limit", 400, false, nil, nil)
		return result, err
	}

	visibility, visibilityErr := normalizeVisibility(req.Visibility)
	if visibilityErr != nil {
		err = newAppError("INVALID_VISIBILITY", "unsupported visibility", 400, false, visibilityErr, nil)
		return result, err
	}

	stageResult, stageErr := s.StageImages(req.Images)
	if stageErr != nil {
		err = stageErr
		return result, err
	}
	result.StagedImagePaths = copyStrings(stageResult.StagedImagePaths)
	result.ImagePath = firstString(stageResult.StagedImagePaths)

	scheduleTime, scheduleErr := validateScheduleAt(req.ScheduleAt)
	if scheduleErr != nil {
		err = scheduleErr
		return result, err
	}

	content := xiaohongshu.PublishImageContent{
		Title:        req.Title,
		Content:      req.Content,
		Tags:         req.Tags,
		ImagePaths:   stageResult.StagedImagePaths,
		ScheduleTime: scheduleTime,
		IsOriginal:   req.IsOriginal,
		Visibility:   visibility,
		Products:     req.Products,
	}

	if err = s.publishContent(ctx, content); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"task_id":    result.TaskID,
			"batch_id":   result.BatchID,
			"tenant_id":  result.TenantID,
			"account_id": result.AccountID,
			"title":      content.Title,
		}).Error("publish content failed")
		return result, err
	}

	result.OK = true
	result.Status = "succeeded"
	return result, nil
}

func (s *XiaohongshuService) StageImages(images []string) (*StageImagesResponse, error) {
	if len(images) == 0 {
		return nil, newAppError("IMAGES_REQUIRED", "images are required", 400, false, nil, nil)
	}

	processor := downloader.NewImageProcessor()
	stagedPaths, err := processor.ProcessImages(images)
	if err != nil {
		return nil, newAppError("IMAGE_STAGING_FAILED", "failed to stage images for publish", 400, false, err, map[string]any{
			"images": images,
		})
	}

	return &StageImagesResponse{
		OK:               true,
		ImagePaths:       copyStrings(images),
		StagedImagePaths: copyStrings(stagedPaths),
		Count:            len(stagedPaths),
	}, nil
}

func (s *XiaohongshuService) publishContent(ctx context.Context, content xiaohongshu.PublishImageContent) error {
	return s.withAccountPage(ctx, func(page *rod.Page, _ *AccountSession) error {
		action, err := xiaohongshu.NewPublishImageAction(page)
		if err != nil {
			return err
		}
		return action.Publish(ctx, content)
	})
}

func (s *XiaohongshuService) PublishVideo(ctx context.Context, req *PublishVideoRequest) (result *PublishExecutionResult, err error) {
	scope := AccountScopeFromContext(ctx)
	meta := operationMetadataFromContext(ctx)
	startedAt := time.Now().UTC()

	result = newPublishExecutionResult(OperationPublishVideo, scope, req.Title, req.Content, firstNonEmpty(req.TaskID, meta.TaskID), firstNonEmpty(req.BatchID, meta.BatchID))
	result.VideoPath = req.Video
	result.PublishStartAt = formatTime(startedAt)

	defer func() {
		s.finishPublishExecution(scope, startedAt, result, err)
	}()

	if xhsutil.CalcTitleLength(req.Title) > 20 {
		err = newAppError("TITLE_TOO_LONG", "title length exceeds limit", 400, false, nil, nil)
		return result, err
	}

	visibility, visibilityErr := normalizeVisibility(req.Visibility)
	if visibilityErr != nil {
		err = newAppError("INVALID_VISIBILITY", "unsupported visibility", 400, false, visibilityErr, nil)
		return result, err
	}
	if req.Video == "" {
		err = newAppError("VIDEO_REQUIRED", "video path is required", 400, false, nil, nil)
		return result, err
	}
	if _, statErr := os.Stat(req.Video); statErr != nil {
		err = newAppError("VIDEO_INACCESSIBLE", "video file inaccessible", 400, false, statErr, map[string]any{
			"video_path": req.Video,
		})
		return result, err
	}

	scheduleTime, scheduleErr := validateScheduleAt(req.ScheduleAt)
	if scheduleErr != nil {
		err = scheduleErr
		return result, err
	}

	content := xiaohongshu.PublishVideoContent{
		Title:        req.Title,
		Content:      req.Content,
		Tags:         req.Tags,
		VideoPath:    req.Video,
		ScheduleTime: scheduleTime,
		Visibility:   visibility,
		Products:     req.Products,
	}

	if err = s.publishVideo(ctx, content); err != nil {
		return result, err
	}

	result.OK = true
	result.Status = "succeeded"
	return result, nil
}

func (s *XiaohongshuService) publishVideo(ctx context.Context, content xiaohongshu.PublishVideoContent) error {
	return s.withAccountPage(ctx, func(page *rod.Page, _ *AccountSession) error {
		action, err := xiaohongshu.NewPublishVideoAction(page)
		if err != nil {
			return err
		}
		return action.PublishVideo(ctx, content)
	})
}

func (s *XiaohongshuService) ListFeeds(ctx context.Context) (*FeedsListResponse, error) {
	var feeds []xiaohongshu.Feed
	if err := s.withAccountPage(ctx, func(page *rod.Page, _ *AccountSession) error {
		action := xiaohongshu.NewFeedsListAction(page)
		f, err := action.GetFeedsList(ctx)
		if err != nil {
			return err
		}
		feeds = f
		return nil
	}); err != nil {
		return nil, err
	}

	return &FeedsListResponse{Feeds: feeds, Count: len(feeds)}, nil
}

func (s *XiaohongshuService) SearchFeeds(ctx context.Context, keyword string, filters ...xiaohongshu.FilterOption) (*FeedsListResponse, error) {
	var feeds []xiaohongshu.Feed
	if err := s.withAccountPage(ctx, func(page *rod.Page, _ *AccountSession) error {
		action := xiaohongshu.NewSearchAction(page)
		f, err := action.Search(ctx, keyword, filters...)
		if err != nil {
			return err
		}
		feeds = f
		return nil
	}); err != nil {
		return nil, err
	}

	return &FeedsListResponse{Feeds: feeds, Count: len(feeds)}, nil
}

func (s *XiaohongshuService) GetFeedDetail(ctx context.Context, feedID, xsecToken string, loadAllComments bool) (*FeedDetailResponse, error) {
	return s.GetFeedDetailWithConfig(ctx, feedID, xsecToken, loadAllComments, xiaohongshu.DefaultCommentLoadConfig())
}

func (s *XiaohongshuService) GetFeedDetailWithConfig(ctx context.Context, feedID, xsecToken string, loadAllComments bool, cfg xiaohongshu.CommentLoadConfig) (*FeedDetailResponse, error) {
	var result any
	if err := s.withAccountPage(ctx, func(page *rod.Page, _ *AccountSession) error {
		action := xiaohongshu.NewFeedDetailAction(page)
		data, err := action.GetFeedDetailWithConfig(ctx, feedID, xsecToken, loadAllComments, cfg)
		if err != nil {
			return err
		}
		result = data
		return nil
	}); err != nil {
		return nil, err
	}

	return &FeedDetailResponse{FeedID: feedID, Data: result}, nil
}

func (s *XiaohongshuService) UserProfile(ctx context.Context, userID, xsecToken string) (*UserProfileResponse, error) {
	var result *xiaohongshu.UserProfileResponse
	if err := s.withAccountPage(ctx, func(page *rod.Page, _ *AccountSession) error {
		action := xiaohongshu.NewUserProfileAction(page)
		data, err := action.UserProfile(ctx, userID, xsecToken)
		if err != nil {
			return err
		}
		result = data
		return nil
	}); err != nil {
		return nil, err
	}

	return &UserProfileResponse{
		UserBasicInfo: result.UserBasicInfo,
		Interactions:  result.Interactions,
		Feeds:         result.Feeds,
	}, nil
}

func (s *XiaohongshuService) PostCommentToFeed(ctx context.Context, feedID, xsecToken, content string) (*PostCommentResponse, error) {
	if err := s.withAccountPage(ctx, func(page *rod.Page, _ *AccountSession) error {
		action := xiaohongshu.NewCommentFeedAction(page)
		return action.PostComment(ctx, feedID, xsecToken, content)
	}); err != nil {
		return nil, err
	}

	return &PostCommentResponse{FeedID: feedID, Success: true, Message: "comment posted"}, nil
}

func (s *XiaohongshuService) LikeFeed(ctx context.Context, feedID, xsecToken string) (*ActionResult, error) {
	if err := s.withAccountPage(ctx, func(page *rod.Page, _ *AccountSession) error {
		action := xiaohongshu.NewLikeAction(page)
		return action.Like(ctx, feedID, xsecToken)
	}); err != nil {
		return nil, err
	}
	return &ActionResult{FeedID: feedID, Success: true, Message: "liked"}, nil
}

func (s *XiaohongshuService) UnlikeFeed(ctx context.Context, feedID, xsecToken string) (*ActionResult, error) {
	if err := s.withAccountPage(ctx, func(page *rod.Page, _ *AccountSession) error {
		action := xiaohongshu.NewLikeAction(page)
		return action.Unlike(ctx, feedID, xsecToken)
	}); err != nil {
		return nil, err
	}
	return &ActionResult{FeedID: feedID, Success: true, Message: "unliked"}, nil
}

func (s *XiaohongshuService) FavoriteFeed(ctx context.Context, feedID, xsecToken string) (*ActionResult, error) {
	if err := s.withAccountPage(ctx, func(page *rod.Page, _ *AccountSession) error {
		action := xiaohongshu.NewFavoriteAction(page)
		return action.Favorite(ctx, feedID, xsecToken)
	}); err != nil {
		return nil, err
	}
	return &ActionResult{FeedID: feedID, Success: true, Message: "favorited"}, nil
}

func (s *XiaohongshuService) UnfavoriteFeed(ctx context.Context, feedID, xsecToken string) (*ActionResult, error) {
	if err := s.withAccountPage(ctx, func(page *rod.Page, _ *AccountSession) error {
		action := xiaohongshu.NewFavoriteAction(page)
		return action.Unfavorite(ctx, feedID, xsecToken)
	}); err != nil {
		return nil, err
	}
	return &ActionResult{FeedID: feedID, Success: true, Message: "unfavorited"}, nil
}

func (s *XiaohongshuService) ReplyCommentToFeed(ctx context.Context, feedID, xsecToken, commentID, userID, content string) (*ReplyCommentResponse, error) {
	if err := s.withAccountPage(ctx, func(page *rod.Page, _ *AccountSession) error {
		action := xiaohongshu.NewCommentFeedAction(page)
		return action.ReplyToComment(ctx, feedID, xsecToken, commentID, userID, content)
	}); err != nil {
		return nil, err
	}

	return &ReplyCommentResponse{
		FeedID:          feedID,
		TargetCommentID: commentID,
		TargetUserID:    userID,
		Success:         true,
		Message:         "reply posted",
	}, nil
}

func (s *XiaohongshuService) GetMyProfile(ctx context.Context) (*UserProfileResponse, error) {
	var result *xiaohongshu.UserProfileResponse
	if err := s.withAccountPage(ctx, func(page *rod.Page, _ *AccountSession) error {
		action := xiaohongshu.NewUserProfileAction(page)
		data, err := action.GetMyProfileViaSidebar(ctx)
		if err != nil {
			return err
		}
		result = data
		return nil
	}); err != nil {
		return nil, err
	}

	return &UserProfileResponse{
		UserBasicInfo: result.UserBasicInfo,
		Interactions:  result.Interactions,
		Feeds:         result.Feeds,
	}, nil
}

func (s *XiaohongshuService) acquireSession(ctx context.Context) (*AccountSession, error) {
	scope := AccountScopeFromContext(ctx)
	return s.accountManager.Acquire(ctx, scope)
}

func (s *XiaohongshuService) withAccountPage(ctx context.Context, fn func(*rod.Page, *AccountSession) error) error {
	session, err := s.acquireSession(ctx)
	if err != nil {
		return err
	}
	releaseBrowserSlot, err := s.browserGuard.Acquire(ctx)
	if err != nil {
		session.Release()
		return err
	}

	b := newBrowserForSession(session)
	s.metrics.BrowserSessionStarted()

	page := b.NewPage()
	var closeOnce sync.Once
	closeResources := func() {
		closeOnce.Do(func() {
			_ = page.Close()
			b.Close()
			session.Release()
			releaseBrowserSlot()
			s.metrics.BrowserSessionEnded()
		})
	}
	defer closeResources()

	done := make(chan struct{})
	defer close(done)

	go func() {
		select {
		case <-ctx.Done():
			closeResources()
		case <-done:
		}
	}()

	return fn(page, session)
}

func newBrowserForSession(session *AccountSession) *headless_browser.Browser {
	opts := []browser.Option{
		browser.WithBinPath(session.BrowserBin),
		browser.WithCookiePath(session.CookiePath),
	}
	if session.Proxy != "" {
		opts = append(opts, browser.WithProxy(session.Proxy))
	}
	return browser.NewBrowser(session.Headless, opts...)
}

func saveCookiesToPath(page *rod.Page, cookiePath string) error {
	cks, err := page.Browser().GetCookies()
	if err != nil {
		return err
	}

	data, err := json.Marshal(cks)
	if err != nil {
		return err
	}

	cookieLoader := cookies.NewLoadCookie(cookiePath)
	return cookieLoader.SaveCookies(data)
}

func accountLabelFromSession(session *AccountSession) string {
	if session == nil {
		return ""
	}
	return session.TenantID + "/" + session.AccountID
}

func newPublishExecutionResult(op OperationName, scope AccountScope, title, content, taskID, batchID string) *PublishExecutionResult {
	return &PublishExecutionResult{
		Mode:      "sync",
		Operation: string(op),
		TaskID:    ensureTaskID(taskID, "pub"),
		BatchID:   batchID,
		TenantID:  scope.TenantID,
		AccountID: scope.AccountID,
		Title:     title,
		Content:   content,
		Status:    "running",
	}
}

func (s *XiaohongshuService) finishPublishExecution(scope AccountScope, startedAt time.Time, result *PublishExecutionResult, err error) {
	if result == nil {
		return
	}

	duration := time.Since(startedAt)
	result.DurationMs = duration.Milliseconds()
	result.PublishEndAt = formatTime(startedAt.Add(duration))

	appErr := classifyError(err)
	if err == nil {
		result.OK = true
		if result.Status == "" || result.Status == "running" {
			result.Status = "succeeded"
		}
		s.accountManager.RecordPublishResult(scope, nil)
		s.metrics.RecordPublish(startedAt, duration, true)
		logrus.WithFields(logrus.Fields{
			"operation":          result.Operation,
			"task_id":            result.TaskID,
			"batch_id":           result.BatchID,
			"tenant_id":          result.TenantID,
			"account_id":         result.AccountID,
			"title":              result.Title,
			"duration_ms":        result.DurationMs,
			"staged_image_paths": result.StagedImagePaths,
			"video_path":         result.VideoPath,
			"publish_start_at":   result.PublishStartAt,
			"publish_end_at":     result.PublishEndAt,
		}).Info("publish operation completed")
		return
	}

	result.OK = false
	result.Status = "failed"
	result.ErrorCode = appErr.Code
	result.ErrorMessage = appErr.Error()
	result.Retryable = appErr.Retryable
	s.accountManager.RecordPublishResult(scope, appErr)
	s.metrics.RecordPublish(startedAt, duration, false)
	logrus.WithError(err).WithFields(logrus.Fields{
		"operation":        result.Operation,
		"task_id":          result.TaskID,
		"batch_id":         result.BatchID,
		"tenant_id":        result.TenantID,
		"account_id":       result.AccountID,
		"title":            result.Title,
		"duration_ms":      result.DurationMs,
		"error_code":       result.ErrorCode,
		"retryable":        result.Retryable,
		"publish_start_at": result.PublishStartAt,
		"publish_end_at":   result.PublishEndAt,
	}).Warn("publish operation failed")
}

func validateScheduleAt(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}

	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, newAppError("INVALID_SCHEDULE_AT", "invalid schedule_at format", 400, false, err, nil)
	}

	now := time.Now()
	minTime := now.Add(1 * time.Hour)
	maxTime := now.Add(14 * 24 * time.Hour)

	if t.Before(minTime) {
		return nil, newAppError("SCHEDULE_TOO_SOON", "schedule_at must be at least 1 hour later", 400, false, nil, map[string]any{
			"schedule_at": raw,
			"earliest":    minTime.Format(time.RFC3339),
		})
	}
	if t.After(maxTime) {
		return nil, newAppError("SCHEDULE_TOO_LATE", "schedule_at must be within 14 days", 400, false, nil, map[string]any{
			"schedule_at": raw,
			"latest":      maxTime.Format(time.RFC3339),
		})
	}

	return &t, nil
}

func ensureTaskID(taskID, prefix string) string {
	if taskID != "" {
		return taskID
	}
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func copyStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	return append([]string(nil), items...)
}

func firstString(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[0]
}

func normalizeVisibility(input string) (string, error) {
	const (
		visibilityPublic  = "\u516c\u5f00\u53ef\u89c1"
		visibilityPrivate = "\u4ec5\u81ea\u5df1\u53ef\u89c1"
		visibilityFriends = "\u4ec5\u4e92\u5173\u597d\u53cb\u53ef\u89c1"
	)

	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return visibilityPublic, nil
	}

	switch trimmed {
	case visibilityPublic:
		return visibilityPublic, nil
	case visibilityPrivate:
		return visibilityPrivate, nil
	case visibilityFriends:
		return visibilityFriends, nil
	}

	normalized := strings.ToLower(trimmed)
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "")
	normalized = replacer.Replace(normalized)

	switch normalized {
	case "public", "open", "everyone", "all", "publicvisible":
		return visibilityPublic, nil
	case "private", "self", "selfonly", "onlyme", "me":
		return visibilityPrivate, nil
	case "friendsonly", "friends", "mutual", "mutualfollow", "mutualfollows":
		return visibilityFriends, nil
	default:
		return "", fmt.Errorf("unsupported visibility: %s, supported values: public/self-only/friends-only or 公开可见/仅自己可见/仅互关好友可见", input)
	}
}

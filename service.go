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
	loginWatchers  map[string]*loginWatcher
	loginMu        sync.Mutex
}

type loginWatcher struct {
	img       string
	timeout   time.Duration
	startedAt time.Time
	cancel    context.CancelFunc
	ready     chan struct{}
	readyOnce sync.Once
	err       error
	loggedIn  bool
}

func NewXiaohongshuService() (*XiaohongshuService, error) {
	manager, err := NewAccountManagerFromEnv()
	if err != nil {
		return nil, err
	}
	return &XiaohongshuService{
		accountManager: manager,
		loginWatchers:  make(map[string]*loginWatcher),
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
	scope, err := s.ResolveScope(AccountScopeFromContext(ctx))
	if err != nil {
		return "", err
	}
	ctx = WithAccountScope(ctx, scope)
	s.clearLoginWatcher(scope.Label())

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
	const timeout = 4 * time.Minute

	scope, err := s.ResolveScope(AccountScopeFromContext(ctx))
	if err != nil {
		return nil, err
	}
	ctx = WithAccountScope(ctx, scope)
	watcherKey := scope.Label()

	if watcher, created := s.getOrCreateLoginWatcher(scope, timeout); watcher != nil {
		if !created {
			if err := watcher.waitUntilReady(ctx); err != nil {
				return nil, err
			}
			if watcher.err != nil {
				return nil, watcher.err
			}
			return loginQrcodeResponseFromWatcher(watcher), nil
		}
		defer func() {
			if watcher.err != nil || watcher.loggedIn {
				s.clearLoginWatcherIfMatch(watcherKey, watcher)
			}
		}()

		session, err := s.acquireSession(ctx)
		if err != nil {
			watcher.err = err
			watcher.finishSetup()
			return nil, err
		}

		b := newBrowserForSession(session)
		page := b.NewPage()
		cleanup := func() {
			_ = page.Close()
			b.Close()
			session.Release()
		}

		loginAction := xiaohongshu.NewLogin(page)
		img, loggedIn, err := loginAction.FetchQrcodeImage(ctx)
		if err != nil || loggedIn {
			cleanup()
		}
		if err != nil {
			watcher.err = err
			watcher.finishSetup()
			return nil, err
		}

		watcher.img = img
		if loggedIn {
			watcher.markLoggedIn()
			return &LoginQrcodeResponse{Timeout: "0s", Img: img, IsLoggedIn: true}, nil
		}

		cookiePath := session.CookiePath
		ctxTimeout, cancel := context.WithTimeout(context.Background(), timeout)
		watcher.cancel = cancel
		watcher.finishSetup()
		go func(activeWatcher *loginWatcher) {
			defer cancel()
			defer cleanup()
			defer s.clearLoginWatcherIfMatch(watcherKey, activeWatcher)

			if loginAction.WaitForLogin(ctxTimeout) {
				activeWatcher.markLoggedIn()
				if saveErr := saveCookiesToPath(page, cookiePath); saveErr != nil {
					logrus.WithError(saveErr).WithField("cookie_path", cookiePath).Error("failed to save cookies")
				}
			}
		}(watcher)
		return loginQrcodeResponseFromWatcher(watcher), nil
	}

	return nil, fmt.Errorf("failed to create login watcher")
}

func (s *XiaohongshuService) PublishContent(ctx context.Context, req *PublishRequest) (*PublishExecutionResult, error) {
	startedAt := time.Now().UTC()
	scope := AccountScopeFromContext(ctx)
	result := &PublishExecutionResult{
		Mode:           firstNonEmpty(req.Mode, string(PublishModeSync)),
		TaskID:         ensureTaskID(req.TaskID, "pub"),
		BatchID:        req.BatchID,
		TenantID:       scope.TenantID,
		AccountID:      scope.AccountID,
		Title:          req.Title,
		Content:        req.Content,
		Status:         "running",
		PublishStartAt: formatRFC3339(startedAt),
		ProductBindingResult: ProductBindingResult{
			ProductsRequested: copyStrings(req.Products),
		},
	}

	finalize := func(err error) (*PublishExecutionResult, error) {
		finishedAt := time.Now().UTC()
		result.PublishEndAt = formatRFC3339(finishedAt)
		result.DurationMs = finishedAt.Sub(startedAt).Milliseconds()
		if err != nil {
			if result.ErrorCode == "" {
				result.ErrorCode = "PUBLISH_FAILED"
			}
			if result.ErrorMessage == "" {
				result.ErrorMessage = err.Error()
			}
			result.OK = false
			if result.Status == "running" {
				result.Status = "failed"
			}
			return result, err
		}
		result.OK = true
		if result.Status == "running" {
			result.Status = "succeeded"
		}
		return result, nil
	}

	if xhsutil.CalcTitleLength(req.Title) > 20 {
		result.ErrorCode = "TITLE_TOO_LONG"
		return finalize(fmt.Errorf("title length exceeds limit"))
	}

	visibility, err := normalizeVisibility(req.Visibility)
	if err != nil {
		result.ErrorCode = "INVALID_VISIBILITY"
		return finalize(err)
	}

	beforeSnapshot, snapshotErr := s.captureMyFeedSnapshot(ctx)
	if snapshotErr != nil {
		beforeSnapshot = publishFeedSnapshot{}
		result.BackfillStatus = string(PublishBackfillPending)
		result.BackfillReason = "failed to capture pre-publish snapshot: " + snapshotErr.Error()
	}

	imagePaths, cleanupResult, err := s.processImagesForPublish(req.Images)
	if err != nil {
		result.PublishCleanupResult = cleanupResult
		result.ErrorCode = "IMAGE_PROCESS_FAILED"
		return finalize(err)
	}
	defer func() {
		cleanupResult = cleanupTempFiles(cleanupResult.Paths)
		result.PublishCleanupResult = cleanupResult
	}()

	var scheduleTime *time.Time
	if req.ScheduleAt != "" {
		t, err := time.Parse(time.RFC3339, req.ScheduleAt)
		if err != nil {
			result.ErrorCode = "INVALID_SCHEDULE_AT"
			return finalize(fmt.Errorf("invalid schedule_at format: %w", err))
		}

		now := time.Now()
		minTime := now.Add(1 * time.Hour)
		maxTime := now.Add(14 * 24 * time.Hour)

		if t.Before(minTime) {
			result.ErrorCode = "SCHEDULE_TOO_SOON"
			return finalize(fmt.Errorf("schedule_at must be at least 1 hour later: got %s earliest %s", t.Format("2006-01-02 15:04"), minTime.Format("2006-01-02 15:04")))
		}
		if t.After(maxTime) {
			result.ErrorCode = "SCHEDULE_TOO_LATE"
			return finalize(fmt.Errorf("schedule_at must be within 14 days: got %s latest %s", t.Format("2006-01-02 15:04"), maxTime.Format("2006-01-02 15:04")))
		}

		scheduleTime = &t
	}

	content := xiaohongshu.PublishImageContent{
		Title:        req.Title,
		Content:      req.Content,
		Tags:         req.Tags,
		ImagePaths:   imagePaths,
		ScheduleTime: scheduleTime,
		IsOriginal:   req.IsOriginal,
		Visibility:   visibility,
		Products:     req.Products,
	}

	artifacts, err := s.publishContent(ctx, content)
	if artifacts != nil {
		result.ProductBindingResult = makeProductBindingResult(artifacts.ProductBind)
	}
	if err != nil {
		logrus.WithError(err).WithField("title", content.Title).Error("publish content failed")
		if result.ErrorCode == "" {
			result.ErrorCode = "PUBLISH_FAILED"
		}
		return finalize(err)
	}

	result.Status = "succeeded"
	if scheduleTime != nil {
		result.BackfillStatus = string(PublishBackfillSkipped)
		result.BackfillReason = "scheduled publish does not create an immediately visible note"
		return finalize(nil)
	}

	feed, err := s.waitForPublishedFeed(ctx, beforeSnapshot, req.Title)
	if err != nil {
		logrus.WithError(err).WithField("title", req.Title).Warn("publish succeeded but entity backfill did not resolve")
		result.BackfillStatus = string(PublishBackfillPending)
		result.BackfillReason = err.Error()
		return finalize(nil)
	}

	detail, detailErr := s.getFeedDetailInternal(ctx, feed.ID, feed.XsecToken)
	if detailErr != nil {
		logrus.WithError(detailErr).WithField("feed_id", feed.ID).Warn("publish entity found but detail fetch failed")
	}
	applyPublishEntity(result, feed, detail)
	result.BackfillStatus = string(PublishBackfillResolved)

	return finalize(nil)
}

func (s *XiaohongshuService) processImages(images []string) ([]string, error) {
	processor := downloader.NewImageProcessor()
	return processor.ProcessImages(images)
}

func (s *XiaohongshuService) processImagesForPublish(images []string) ([]string, PublishCleanupResult, error) {
	localPaths, err := s.processImages(images)
	if err != nil {
		return nil, PublishCleanupResult{}, err
	}

	cleanupPaths := make([]string, 0, len(images))
	for idx, image := range images {
		if idx >= len(localPaths) {
			break
		}
		if downloader.IsImageURL(image) {
			cleanupPaths = append(cleanupPaths, localPaths[idx])
		}
	}

	result := PublishCleanupResult{Status: "skipped"}
	if len(cleanupPaths) > 0 {
		result.Status = "pending"
		result.Paths = cleanupPaths
	}
	return localPaths, result, nil
}

func (s *XiaohongshuService) publishContent(ctx context.Context, content xiaohongshu.PublishImageContent) (*xiaohongshu.PublishArtifacts, error) {
	var artifacts *xiaohongshu.PublishArtifacts
	err := s.withAccountPage(ctx, func(page *rod.Page, _ *AccountSession) error {
		action, err := xiaohongshu.NewPublishImageAction(page)
		if err != nil {
			return err
		}
		artifacts, err = action.Publish(ctx, content)
		return err
	})
	return artifacts, err
}

func (s *XiaohongshuService) PublishVideo(ctx context.Context, req *PublishVideoRequest) (*PublishExecutionResult, error) {
	startedAt := time.Now().UTC()
	scope := AccountScopeFromContext(ctx)
	result := &PublishExecutionResult{
		Mode:           firstNonEmpty(req.Mode, string(PublishModeSync)),
		TaskID:         ensureTaskID(req.TaskID, "pub"),
		BatchID:        req.BatchID,
		TenantID:       scope.TenantID,
		AccountID:      scope.AccountID,
		Title:          req.Title,
		Content:        req.Content,
		Status:         "running",
		PublishStartAt: formatRFC3339(startedAt),
		ProductBindingResult: ProductBindingResult{
			ProductsRequested: copyStrings(req.Products),
		},
	}

	finalize := func(err error) (*PublishExecutionResult, error) {
		finishedAt := time.Now().UTC()
		result.PublishEndAt = formatRFC3339(finishedAt)
		result.DurationMs = finishedAt.Sub(startedAt).Milliseconds()
		if err != nil {
			if result.ErrorCode == "" {
				result.ErrorCode = "PUBLISH_VIDEO_FAILED"
			}
			if result.ErrorMessage == "" {
				result.ErrorMessage = err.Error()
			}
			result.OK = false
			if result.Status == "running" {
				result.Status = "failed"
			}
			return result, err
		}
		result.OK = true
		if result.Status == "running" {
			result.Status = "succeeded"
		}
		return result, nil
	}

	if xhsutil.CalcTitleLength(req.Title) > 20 {
		result.ErrorCode = "TITLE_TOO_LONG"
		return finalize(fmt.Errorf("title length exceeds limit"))
	}

	visibility, err := normalizeVisibility(req.Visibility)
	if err != nil {
		result.ErrorCode = "INVALID_VISIBILITY"
		return finalize(err)
	}
	if req.Video == "" {
		result.ErrorCode = "VIDEO_REQUIRED"
		return finalize(fmt.Errorf("video path is required"))
	}
	if _, err := os.Stat(req.Video); err != nil {
		result.ErrorCode = "VIDEO_INACCESSIBLE"
		return finalize(fmt.Errorf("video file inaccessible: %w", err))
	}

	beforeSnapshot, snapshotErr := s.captureMyFeedSnapshot(ctx)
	if snapshotErr != nil {
		beforeSnapshot = publishFeedSnapshot{}
		result.BackfillStatus = string(PublishBackfillPending)
		result.BackfillReason = "failed to capture pre-publish snapshot: " + snapshotErr.Error()
	}

	var scheduleTime *time.Time
	if req.ScheduleAt != "" {
		t, err := time.Parse(time.RFC3339, req.ScheduleAt)
		if err != nil {
			result.ErrorCode = "INVALID_SCHEDULE_AT"
			return finalize(fmt.Errorf("invalid schedule_at format: %w", err))
		}

		now := time.Now()
		minTime := now.Add(1 * time.Hour)
		maxTime := now.Add(14 * 24 * time.Hour)

		if t.Before(minTime) {
			result.ErrorCode = "SCHEDULE_TOO_SOON"
			return finalize(fmt.Errorf("schedule_at must be at least 1 hour later: got %s earliest %s", t.Format("2006-01-02 15:04"), minTime.Format("2006-01-02 15:04")))
		}
		if t.After(maxTime) {
			result.ErrorCode = "SCHEDULE_TOO_LATE"
			return finalize(fmt.Errorf("schedule_at must be within 14 days: got %s latest %s", t.Format("2006-01-02 15:04"), maxTime.Format("2006-01-02 15:04")))
		}

		scheduleTime = &t
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

	artifacts, err := s.publishVideo(ctx, content)
	if artifacts != nil {
		result.ProductBindingResult = makeProductBindingResult(artifacts.ProductBind)
	}
	if err != nil {
		return finalize(err)
	}

	result.Status = "succeeded"
	if scheduleTime != nil {
		result.BackfillStatus = string(PublishBackfillSkipped)
		result.BackfillReason = "scheduled publish does not create an immediately visible note"
		return finalize(nil)
	}

	feed, err := s.waitForPublishedFeed(ctx, beforeSnapshot, req.Title)
	if err != nil {
		logrus.WithError(err).WithField("title", req.Title).Warn("video publish succeeded but entity backfill did not resolve")
		result.BackfillStatus = string(PublishBackfillPending)
		result.BackfillReason = err.Error()
		return finalize(nil)
	}

	detail, detailErr := s.getFeedDetailInternal(ctx, feed.ID, feed.XsecToken)
	if detailErr != nil {
		logrus.WithError(detailErr).WithField("feed_id", feed.ID).Warn("video publish entity found but detail fetch failed")
	}
	applyPublishEntity(result, feed, detail)
	result.BackfillStatus = string(PublishBackfillResolved)

	return finalize(nil)
}

func (s *XiaohongshuService) publishVideo(ctx context.Context, content xiaohongshu.PublishVideoContent) (*xiaohongshu.PublishArtifacts, error) {
	var artifacts *xiaohongshu.PublishArtifacts
	err := s.withAccountPage(ctx, func(page *rod.Page, _ *AccountSession) error {
		action, err := xiaohongshu.NewPublishVideoAction(page)
		if err != nil {
			return err
		}
		artifacts, err = action.PublishVideo(ctx, content)
		return err
	})
	return artifacts, err
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

func (s *XiaohongshuService) ListRecentPublishedNotes(ctx context.Context, req *RecentPublishedNotesRequest) (*RecentPublishedNotesResponse, error) {
	profile, err := s.GetMyProfile(ctx)
	if err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	var sinceTime time.Time
	if strings.TrimSpace(req.SinceTime) != "" {
		sinceTime, err = time.Parse(time.RFC3339, req.SinceTime)
		if err != nil {
			return nil, fmt.Errorf("invalid since_time format: %w", err)
		}
	}

	notes := make([]RecentPublishedNote, 0, limit)
	keyword := strings.TrimSpace(req.TitleKeyword)
	for _, feed := range profile.Feeds {
		if len(notes) >= limit {
			break
		}
		title := strings.TrimSpace(feed.NoteCard.DisplayTitle)
		if keyword != "" && !strings.Contains(title, keyword) {
			continue
		}

		note := RecentPublishedNote{
			NoteID:    feed.ID,
			NoteURL:   buildFeedNoteURL(feed.ID, feed.XsecToken),
			FeedID:    feed.ID,
			XsecToken: feed.XsecToken,
			Title:     title,
		}

		detail, detailErr := s.getFeedDetailInternal(ctx, feed.ID, feed.XsecToken)
		if detailErr == nil && detail != nil {
			if detail.Note.NoteID != "" {
				note.NoteID = detail.Note.NoteID
			}
			note.PublishTime = unixToRFC3339(detail.Note.Time)
		}

		if !sinceTime.IsZero() && note.PublishTime != "" {
			publishedAt, parseErr := time.Parse(time.RFC3339, note.PublishTime)
			if parseErr == nil && publishedAt.Before(sinceTime) {
				continue
			}
		}

		notes = append(notes, note)
	}

	return &RecentPublishedNotesResponse{Notes: notes, Count: len(notes)}, nil
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
	defer session.Release()

	b := newBrowserForSession(session)
	defer b.Close()

	page := b.NewPage()
	defer page.Close()

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

func (s *XiaohongshuService) getOrCreateLoginWatcher(scope AccountScope, timeout time.Duration) (*loginWatcher, bool) {
	key := scope.Label()
	if key == "" {
		return nil, false
	}

	s.loginMu.Lock()
	defer s.loginMu.Unlock()

	if watcher := s.loginWatchers[key]; watcher != nil {
		if remainingLoginWatcherTimeout(watcher.startedAt, watcher.timeout) > 0 {
			return watcher, false
		}
		delete(s.loginWatchers, key)
		if watcher.cancel != nil {
			watcher.cancel()
		}
	}

	watcher := &loginWatcher{
		timeout:   timeout,
		startedAt: time.Now(),
		ready:     make(chan struct{}),
	}
	s.loginWatchers[key] = watcher
	return watcher, true
}

func (w *loginWatcher) markLoggedIn() {
	if w == nil {
		return
	}
	w.loggedIn = true
	w.finishSetup()
}

func (w *loginWatcher) finishSetup() {
	if w == nil {
		return
	}
	w.readyOnce.Do(func() {
		close(w.ready)
	})
}

func (w *loginWatcher) waitUntilReady(ctx context.Context) error {
	if w == nil || w.ready == nil {
		return nil
	}
	select {
	case <-w.ready:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func loginQrcodeResponseFromWatcher(watcher *loginWatcher) *LoginQrcodeResponse {
	if watcher == nil {
		return &LoginQrcodeResponse{}
	}
	if watcher.loggedIn {
		return &LoginQrcodeResponse{Timeout: "0s", Img: watcher.img, IsLoggedIn: true}
	}
	return &LoginQrcodeResponse{
		Timeout:    remainingLoginWatcherTimeout(watcher.startedAt, watcher.timeout).String(),
		Img:        watcher.img,
		IsLoggedIn: false,
	}
}

func (s *XiaohongshuService) getLoginWatcher(key string) *loginWatcher {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}

	s.loginMu.Lock()
	watcher := s.loginWatchers[key]
	if watcher == nil {
		s.loginMu.Unlock()
		return nil
	}
	if remainingLoginWatcherTimeout(watcher.startedAt, watcher.timeout) > 0 {
		s.loginMu.Unlock()
		return watcher
	}
	delete(s.loginWatchers, key)
	cancel := watcher.cancel
	s.loginMu.Unlock()

	if cancel != nil {
		cancel()
	}
	return nil
}

func (s *XiaohongshuService) setLoginWatcher(key string, watcher *loginWatcher) {
	key = strings.TrimSpace(key)
	if key == "" || watcher == nil {
		return
	}

	s.loginMu.Lock()
	previous := s.loginWatchers[key]
	s.loginWatchers[key] = watcher
	s.loginMu.Unlock()

	if previous != nil && previous != watcher && previous.cancel != nil {
		previous.cancel()
	}
}

func (s *XiaohongshuService) clearLoginWatcher(key string) {
	s.clearLoginWatcherIfMatch(key, nil)
}

func (s *XiaohongshuService) clearLoginWatcherIfMatch(key string, watcher *loginWatcher) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}

	var cancel context.CancelFunc
	s.loginMu.Lock()
	current := s.loginWatchers[key]
	if current != nil && (watcher == nil || current == watcher) {
		delete(s.loginWatchers, key)
		cancel = current.cancel
	}
	s.loginMu.Unlock()

	if cancel != nil {
		cancel()
	}
}

func remainingLoginWatcherTimeout(startedAt time.Time, timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return 0
	}
	if startedAt.IsZero() {
		return timeout
	}
	remaining := timeout - time.Since(startedAt)
	if remaining <= 0 {
		return 0
	}
	return remaining
}

func accountLabelFromSession(session *AccountSession) string {
	if session == nil {
		return ""
	}
	return AccountScope{TenantID: session.TenantID, AccountID: session.AccountID}.Label()
}

func cleanupTempFiles(paths []string) PublishCleanupResult {
	result := PublishCleanupResult{
		Status: "skipped",
		Paths:  copyStrings(paths),
	}
	if len(paths) == 0 {
		return result
	}

	result.Status = "completed"
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			result.Status = "partial"
			result.Errors = append(result.Errors, err.Error())
		}
	}
	if len(result.Errors) == len(paths) {
		result.Status = "failed"
	}
	return result
}

func makeProductBindingResult(report xiaohongshu.ProductBindReport) ProductBindingResult {
	return ProductBindingResult{
		Status:            report.Status,
		Count:             report.Count,
		ProductsRequested: copyStrings(report.ProductsRequested),
		ProductsResolved:  copyStrings(report.ProductsResolved),
		ProductsMissing:   copyStrings(report.ProductsMissing),
		VerifyConfidence:  report.VerifyConfidence,
	}
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

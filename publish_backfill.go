package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/hekaixin66-sketch/xiaohongshuritter/xiaohongshu"
)

const (
	defaultPublishBackfillTimeout  = 90 * time.Second
	defaultPublishBackfillInterval = 3 * time.Second
)

type publishFeedSnapshot map[string]xiaohongshu.Feed

func formatRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func ensureTaskID(taskID, prefix string) string {
	if strings.TrimSpace(taskID) != "" {
		return strings.TrimSpace(taskID)
	}
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func copyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func buildFeedNoteURL(feedID, xsecToken string) string {
	if feedID == "" || xsecToken == "" {
		return ""
	}
	return fmt.Sprintf("https://www.xiaohongshu.com/explore/%s?xsec_token=%s&xsec_source=pc_feed", feedID, xsecToken)
}

func snapshotFeeds(feeds []xiaohongshu.Feed) publishFeedSnapshot {
	result := make(publishFeedSnapshot, len(feeds))
	for _, feed := range feeds {
		result[feed.ID] = feed
	}
	return result
}

func findNewPublishedFeed(before publishFeedSnapshot, after []xiaohongshu.Feed, title string) *xiaohongshu.Feed {
	var fallback *xiaohongshu.Feed
	normalizedTitle := strings.TrimSpace(title)

	for i := range after {
		feed := after[i]
		if _, exists := before[feed.ID]; exists {
			continue
		}
		if fallback == nil {
			fallback = &after[i]
		}
		if normalizedTitle != "" && strings.TrimSpace(feed.NoteCard.DisplayTitle) == normalizedTitle {
			return &after[i]
		}
	}

	return fallback
}

func unixToRFC3339(value int64) string {
	if value <= 0 {
		return ""
	}
	if value > 1_000_000_000_000 {
		return time.UnixMilli(value).UTC().Format(time.RFC3339)
	}
	return time.Unix(value, 0).UTC().Format(time.RFC3339)
}

func applyPublishEntity(result *PublishExecutionResult, feed *xiaohongshu.Feed, detail *xiaohongshu.FeedDetailResponse) {
	if result == nil || feed == nil {
		return
	}

	result.FeedID = feed.ID
	result.XsecToken = feed.XsecToken
	result.NoteURL = buildFeedNoteURL(feed.ID, feed.XsecToken)
	if detail != nil {
		if detail.Note.NoteID != "" {
			result.NoteID = detail.Note.NoteID
		}
		if result.Title == "" {
			result.Title = detail.Note.Title
		}
		result.PublishedAt = unixToRFC3339(detail.Note.Time)
	}
	if result.NoteID == "" {
		result.NoteID = feed.ID
	}
	result.PublishVerificationResult = evaluatePublishVerification(detail, result.ProductBindingResult)
}

func (s *XiaohongshuService) captureMyFeedSnapshot(ctx context.Context) (publishFeedSnapshot, error) {
	profile, err := s.GetMyProfile(ctx)
	if err != nil {
		return nil, err
	}
	return snapshotFeeds(profile.Feeds), nil
}

func (s *XiaohongshuService) waitForPublishedFeed(ctx context.Context, before publishFeedSnapshot, title string) (*xiaohongshu.Feed, error) {
	deadline := time.Now().Add(defaultPublishBackfillTimeout)
	for {
		profile, err := s.GetMyProfile(ctx)
		if err == nil {
			if feed := findNewPublishedFeed(before, profile.Feeds, title); feed != nil {
				return feed, nil
			}
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("published feed not found before timeout")
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(defaultPublishBackfillInterval):
		}
	}
}

func (s *XiaohongshuService) getFeedDetailInternal(ctx context.Context, feedID, xsecToken string) (*xiaohongshu.FeedDetailResponse, error) {
	var result *xiaohongshu.FeedDetailResponse
	if err := s.withAccountPage(ctx, func(page *rod.Page, _ *AccountSession) error {
		action := xiaohongshu.NewFeedDetailAction(page)
		data, err := action.GetFeedDetailWithConfig(ctx, feedID, xsecToken, false, xiaohongshu.DefaultCommentLoadConfig())
		if err != nil {
			return err
		}
		result = data
		return nil
	}); err != nil {
		return nil, err
	}
	return result, nil
}

func evaluatePublishVerification(detail *xiaohongshu.FeedDetailResponse, product ProductBindingResult) PublishVerificationResult {
	result := PublishVerificationResult{
		VerifyStatus: "pending",
		VerifyReason: "entity_backfill_pending",
	}
	if detail == nil {
		return result
	}

	result.PublishVisible = true
	result.CoverVisible = detail.Note.Type == "video" || len(detail.Note.ImageList) > 0

	if len(product.ProductsRequested) == 0 {
		result.VerifyStatus = "verified"
		result.VerifyReason = "publish_visible"
		return result
	}

	switch product.Status {
	case "success":
		result.ProductVisible = boolRef(true)
		result.VerifyStatus = "verified"
		result.VerifyReason = "publish_visible_and_product_bind_success"
	case "failed":
		result.ProductVisible = boolRef(false)
		result.VerifyStatus = "partial"
		result.VerifyReason = "publish_visible_but_product_bind_failed"
	case "partial":
		result.VerifyStatus = "partial"
		result.VerifyReason = "publish_visible_but_product_bind_partial"
	default:
		result.VerifyStatus = "partial"
		result.VerifyReason = "publish_visible_product_visibility_unconfirmed"
	}

	return result
}

func boolRef(v bool) *bool {
	return &v
}

package main

import (
	"context"
	"os"
	"strings"
	"time"
)

type OperationName string

const (
	OperationListAccounts        OperationName = "list_accounts"
	OperationCheckLoginStatus    OperationName = "check_login_status"
	OperationGetLoginQRCode      OperationName = "get_login_qrcode"
	OperationDeleteCookies       OperationName = "delete_cookies"
	OperationPublishContent      OperationName = "publish_content"
	OperationPublishVideo        OperationName = "publish_video"
	OperationPublishContentAsync OperationName = "submit_publish_content_async"
	OperationPublishVideoAsync   OperationName = "submit_publish_video_async"
	OperationPublishJobStatus    OperationName = "get_publish_job_status"
	OperationListFeeds           OperationName = "list_feeds"
	OperationSearchFeeds         OperationName = "search_feeds"
	OperationGetFeedDetail       OperationName = "get_feed_detail"
	OperationUserProfile         OperationName = "user_profile"
	OperationPostComment         OperationName = "post_comment_to_feed"
	OperationReplyComment        OperationName = "reply_comment_in_feed"
	OperationLikeFeed            OperationName = "like_feed"
	OperationFavoriteFeed        OperationName = "favorite_feed"
	OperationStageImagePublish   OperationName = "stage_image_for_publish"
	OperationRuntimeStats        OperationName = "runtime_stats"
)

type operationMetadata struct {
	Name    OperationName
	TaskID  string
	BatchID string
}

type operationMetadataContextKey struct{}

func withOperationMetadata(ctx context.Context, meta operationMetadata) context.Context {
	return context.WithValue(ctx, operationMetadataContextKey{}, meta)
}

func operationMetadataFromContext(ctx context.Context) operationMetadata {
	if meta, ok := ctx.Value(operationMetadataContextKey{}).(operationMetadata); ok {
		return meta
	}
	return operationMetadata{}
}

func timeoutForOperation(op OperationName) time.Duration {
	if override := strings.TrimSpace(os.Getenv("XHS_TIMEOUT_" + strings.ToUpper(string(op)))); override != "" {
		if d, err := time.ParseDuration(override); err == nil && d > 0 {
			return d
		}
	}

	switch op {
	case OperationListAccounts, OperationCheckLoginStatus, OperationDeleteCookies, OperationRuntimeStats,
		OperationPublishContentAsync, OperationPublishVideoAsync, OperationPublishJobStatus:
		return 30 * time.Second
	case OperationGetLoginQRCode, OperationListFeeds, OperationSearchFeeds, OperationGetFeedDetail,
		OperationUserProfile, OperationPostComment, OperationReplyComment, OperationLikeFeed,
		OperationFavoriteFeed, OperationStageImagePublish:
		return 60 * time.Second
	case OperationPublishContent:
		return 300 * time.Second
	case OperationPublishVideo:
		return 300 * time.Second
	default:
		return 60 * time.Second
	}
}

func (s *AppServer) withOperationTimeout(ctx context.Context, op OperationName) (context.Context, context.CancelFunc) {
	timeout := timeoutForOperation(op)
	meta := operationMetadataFromContext(ctx)
	meta.Name = op
	ctx = withOperationMetadata(ctx, meta)

	if timeout <= 0 {
		return context.WithCancel(ctx)
	}

	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining > 0 && remaining <= timeout {
			return context.WithCancel(ctx)
		}
	}

	return context.WithTimeout(ctx, timeout)
}

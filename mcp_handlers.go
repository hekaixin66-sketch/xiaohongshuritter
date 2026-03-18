package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/hekaixin66-sketch/xiaohongshuritter/xiaohongshu"
)

func parseVisibility(args map[string]interface{}) string {
	v, ok := args["visibility"]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func (s *AppServer) handleCheckLoginStatus(ctx context.Context) *MCPToolResult {
	status, err := s.xiaohongshuService.CheckLoginStatus(ctx)
	if err != nil {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "check login status failed: " + err.Error()}}, IsError: true}
	}

	scope := scopeLabel(AccountScopeFromContext(ctx))
	if status.IsLoggedIn {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("logged in (%s)", scope)}}}
	}
	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("not logged in (%s), use get_login_qrcode", scope)}}}
}

func (s *AppServer) handleGetLoginQrcode(ctx context.Context) *MCPToolResult {
	result, err := s.xiaohongshuService.GetLoginQrcode(ctx)
	if err != nil {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "get login qrcode failed: " + err.Error()}}, IsError: true}
	}

	if result.IsLoggedIn {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "already logged in"}}}
	}

	now := time.Now()
	deadline := func() string {
		d, err := time.ParseDuration(result.Timeout)
		if err != nil {
			return now.Format("2006-01-02 15:04:05")
		}
		return now.Add(d).Format("2006-01-02 15:04:05")
	}()

	contents := []MCPContent{
		{Type: "text", Text: "scan qrcode before " + deadline},
		{Type: "image", MimeType: "image/png", Data: strings.TrimPrefix(result.Img, "data:image/png;base64,")},
	}
	return &MCPToolResult{Content: contents}
}

func (s *AppServer) handleDeleteCookies(ctx context.Context) *MCPToolResult {
	cookiePath, err := s.xiaohongshuService.DeleteCookies(ctx)
	if err != nil {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "delete cookies failed: " + err.Error()}}, IsError: true}
	}

	scope := scopeLabel(AccountScopeFromContext(ctx))
	text := fmt.Sprintf("cookies deleted for %s\npath: %s", scope, cookiePath)
	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: text}}}
}

func (s *AppServer) handlePublishContent(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	title, _ := args["title"].(string)
	content, _ := args["content"].(string)
	imagePaths := parseStringSlice(args["images"])
	tags := parseStringSlice(args["tags"])
	products := parseStringSlice(args["products"])
	scheduleAt, _ := args["schedule_at"].(string)
	visibility := parseVisibility(args)
	isOriginal, _ := args["is_original"].(bool)

	req := &PublishRequest{
		Title:      title,
		Content:    content,
		Images:     imagePaths,
		Tags:       tags,
		ScheduleAt: scheduleAt,
		IsOriginal: isOriginal,
		Visibility: visibility,
		Products:   products,
	}

	result, err := s.xiaohongshuService.PublishContent(ctx, req)
	if err != nil {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "publish failed: " + err.Error()}}, IsError: true}
	}

	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("publish success: %+v", result)}}}
}

func (s *AppServer) handlePublishVideo(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	title, _ := args["title"].(string)
	content, _ := args["content"].(string)
	videoPath, _ := args["video"].(string)
	tags := parseStringSlice(args["tags"])
	products := parseStringSlice(args["products"])
	scheduleAt, _ := args["schedule_at"].(string)
	visibility := parseVisibility(args)

	if videoPath == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "publish failed: missing video path"}}, IsError: true}
	}

	req := &PublishVideoRequest{
		Title:      title,
		Content:    content,
		Video:      videoPath,
		Tags:       tags,
		ScheduleAt: scheduleAt,
		Visibility: visibility,
		Products:   products,
	}

	result, err := s.xiaohongshuService.PublishVideo(ctx, req)
	if err != nil {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "publish video failed: " + err.Error()}}, IsError: true}
	}

	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("publish video success: %+v", result)}}}
}

func (s *AppServer) handleListFeeds(ctx context.Context) *MCPToolResult {
	result, err := s.xiaohongshuService.ListFeeds(ctx)
	if err != nil {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "list feeds failed: " + err.Error()}}, IsError: true}
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("list feeds succeeded but marshal failed: %v", err)}}, IsError: true}
	}

	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: string(jsonData)}}}
}

func (s *AppServer) handleSearchFeeds(ctx context.Context, args SearchFeedsArgs) *MCPToolResult {
	if args.Keyword == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "search feeds failed: missing keyword"}}, IsError: true}
	}

	filter := xiaohongshu.FilterOption{
		SortBy:      args.Filters.SortBy,
		NoteType:    args.Filters.NoteType,
		PublishTime: args.Filters.PublishTime,
		SearchScope: args.Filters.SearchScope,
		Location:    args.Filters.Location,
	}

	result, err := s.xiaohongshuService.SearchFeeds(ctx, args.Keyword, filter)
	if err != nil {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "search feeds failed: " + err.Error()}}, IsError: true}
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("search succeeded but marshal failed: %v", err)}}, IsError: true}
	}

	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: string(jsonData)}}}
}

func (s *AppServer) handleGetFeedDetail(ctx context.Context, args map[string]any) *MCPToolResult {
	feedID, ok := args["feed_id"].(string)
	if !ok || feedID == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "get feed detail failed: missing feed_id"}}, IsError: true}
	}

	xsecToken, ok := args["xsec_token"].(string)
	if !ok || xsecToken == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "get feed detail failed: missing xsec_token"}}, IsError: true}
	}

	loadAll := false
	if raw, ok := args["load_all_comments"]; ok {
		switch v := raw.(type) {
		case bool:
			loadAll = v
		case string:
			if parsed, err := strconv.ParseBool(v); err == nil {
				loadAll = parsed
			}
		case float64:
			loadAll = v != 0
		}
	}

	cfg := xiaohongshu.DefaultCommentLoadConfig()
	if raw, ok := args["click_more_replies"]; ok {
		switch v := raw.(type) {
		case bool:
			cfg.ClickMoreReplies = v
		case string:
			if parsed, err := strconv.ParseBool(v); err == nil {
				cfg.ClickMoreReplies = parsed
			}
		}
	}
	if raw, ok := args["max_replies_threshold"]; ok {
		switch v := raw.(type) {
		case float64:
			cfg.MaxRepliesThreshold = int(v)
		case string:
			if parsed, err := strconv.Atoi(v); err == nil {
				cfg.MaxRepliesThreshold = parsed
			}
		case int:
			cfg.MaxRepliesThreshold = v
		}
	}
	if raw, ok := args["max_comment_items"]; ok {
		switch v := raw.(type) {
		case float64:
			cfg.MaxCommentItems = int(v)
		case string:
			if parsed, err := strconv.Atoi(v); err == nil {
				cfg.MaxCommentItems = parsed
			}
		case int:
			cfg.MaxCommentItems = v
		}
	}
	if raw, ok := args["scroll_speed"].(string); ok && raw != "" {
		cfg.ScrollSpeed = raw
	}

	result, err := s.xiaohongshuService.GetFeedDetailWithConfig(ctx, feedID, xsecToken, loadAll, cfg)
	if err != nil {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "get feed detail failed: " + err.Error()}}, IsError: true}
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("get feed detail succeeded but marshal failed: %v", err)}}, IsError: true}
	}

	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: string(jsonData)}}}
}

func (s *AppServer) handleUserProfile(ctx context.Context, args map[string]any) *MCPToolResult {
	userID, ok := args["user_id"].(string)
	if !ok || userID == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "get user profile failed: missing user_id"}}, IsError: true}
	}
	xsecToken, ok := args["xsec_token"].(string)
	if !ok || xsecToken == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "get user profile failed: missing xsec_token"}}, IsError: true}
	}

	result, err := s.xiaohongshuService.UserProfile(ctx, userID, xsecToken)
	if err != nil {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "get user profile failed: " + err.Error()}}, IsError: true}
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("get user profile succeeded but marshal failed: %v", err)}}, IsError: true}
	}

	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: string(jsonData)}}}
}

func (s *AppServer) handleLikeFeed(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	feedID, ok := args["feed_id"].(string)
	if !ok || feedID == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "operation failed: missing feed_id"}}, IsError: true}
	}
	xsecToken, ok := args["xsec_token"].(string)
	if !ok || xsecToken == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "operation failed: missing xsec_token"}}, IsError: true}
	}
	unlike, _ := args["unlike"].(bool)

	var (
		res *ActionResult
		err error
	)
	if unlike {
		res, err = s.xiaohongshuService.UnlikeFeed(ctx, feedID, xsecToken)
	} else {
		res, err = s.xiaohongshuService.LikeFeed(ctx, feedID, xsecToken)
	}
	if err != nil {
		action := "like"
		if unlike {
			action = "unlike"
		}
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: action + " failed: " + err.Error()}}, IsError: true}
	}

	action := "like"
	if unlike {
		action = "unlike"
	}
	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("%s success - Feed ID: %s", action, res.FeedID)}}}
}

func (s *AppServer) handleFavoriteFeed(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	feedID, ok := args["feed_id"].(string)
	if !ok || feedID == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "operation failed: missing feed_id"}}, IsError: true}
	}
	xsecToken, ok := args["xsec_token"].(string)
	if !ok || xsecToken == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "operation failed: missing xsec_token"}}, IsError: true}
	}
	unfavorite, _ := args["unfavorite"].(bool)

	var (
		res *ActionResult
		err error
	)
	if unfavorite {
		res, err = s.xiaohongshuService.UnfavoriteFeed(ctx, feedID, xsecToken)
	} else {
		res, err = s.xiaohongshuService.FavoriteFeed(ctx, feedID, xsecToken)
	}
	if err != nil {
		action := "favorite"
		if unfavorite {
			action = "unfavorite"
		}
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: action + " failed: " + err.Error()}}, IsError: true}
	}

	action := "favorite"
	if unfavorite {
		action = "unfavorite"
	}
	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("%s success - Feed ID: %s", action, res.FeedID)}}}
}

func (s *AppServer) handlePostComment(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	feedID, ok := args["feed_id"].(string)
	if !ok || feedID == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "post comment failed: missing feed_id"}}, IsError: true}
	}
	xsecToken, ok := args["xsec_token"].(string)
	if !ok || xsecToken == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "post comment failed: missing xsec_token"}}, IsError: true}
	}
	content, ok := args["content"].(string)
	if !ok || content == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "post comment failed: missing content"}}, IsError: true}
	}

	result, err := s.xiaohongshuService.PostCommentToFeed(ctx, feedID, xsecToken, content)
	if err != nil {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "post comment failed: " + err.Error()}}, IsError: true}
	}

	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("comment posted - Feed ID: %s", result.FeedID)}}}
}

func (s *AppServer) handleReplyComment(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	feedID, ok := args["feed_id"].(string)
	if !ok || feedID == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "reply comment failed: missing feed_id"}}, IsError: true}
	}
	xsecToken, ok := args["xsec_token"].(string)
	if !ok || xsecToken == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "reply comment failed: missing xsec_token"}}, IsError: true}
	}
	commentID, _ := args["comment_id"].(string)
	userID, _ := args["user_id"].(string)
	if commentID == "" && userID == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "reply comment failed: missing comment_id or user_id"}}, IsError: true}
	}
	content, ok := args["content"].(string)
	if !ok || content == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "reply comment failed: missing content"}}, IsError: true}
	}

	result, err := s.xiaohongshuService.ReplyCommentToFeed(ctx, feedID, xsecToken, commentID, userID, content)
	if err != nil {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "reply comment failed: " + err.Error()}}, IsError: true}
	}

	responseText := fmt.Sprintf("comment replied - Feed ID: %s, Comment ID: %s, User ID: %s", result.FeedID, result.TargetCommentID, result.TargetUserID)
	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: responseText}}}
}

func (s *AppServer) handleListAccounts(ctx context.Context) *MCPToolResult {
	data := map[string]any{
		"config_path": s.xiaohongshuService.AccountConfigPath(),
		"accounts":    s.xiaohongshuService.ListAccounts(),
	}
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		logrus.WithError(err).Error("marshal list accounts failed")
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "list accounts failed: marshal error"}}, IsError: true}
	}
	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: string(jsonData)}}}
}

func parseStringSlice(raw any) []string {
	switch v := raw.(type) {
	case []string:
		return v
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

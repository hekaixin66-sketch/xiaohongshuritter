package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"runtime/debug"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"
)

func boolPtr(b bool) *bool { return &b }

type AccountScopedArgs struct {
	TenantID  string `json:"tenant_id,omitempty" jsonschema:"tenant id, optional"`
	AccountID string `json:"account_id,omitempty" jsonschema:"account id, optional"`
}

func (a AccountScopedArgs) Scope() AccountScope {
	return AccountScope{TenantID: a.TenantID, AccountID: a.AccountID}
}

// PublishContentArgs MCP args for image publish.
type PublishContentArgs struct {
	AccountScopedArgs
	Title      string   `json:"title" jsonschema:"post title"`
	Content    string   `json:"content" jsonschema:"post content"`
	Images     []string `json:"images" jsonschema:"image urls or absolute local file paths"`
	Tags       []string `json:"tags,omitempty" jsonschema:"topic tags"`
	ScheduleAt string   `json:"schedule_at,omitempty" jsonschema:"RFC3339 schedule time, 1 hour to 14 days"`
	IsOriginal bool     `json:"is_original,omitempty" jsonschema:"declare original content"`
	Visibility string   `json:"visibility,omitempty" jsonschema:"visibility level"`
	Products   []string `json:"products,omitempty" jsonschema:"product search keywords"`
}

// PublishVideoArgs MCP args for video publish.
type PublishVideoArgs struct {
	AccountScopedArgs
	Title      string   `json:"title" jsonschema:"post title"`
	Content    string   `json:"content" jsonschema:"post content"`
	Video      string   `json:"video" jsonschema:"absolute local video path"`
	Tags       []string `json:"tags,omitempty" jsonschema:"topic tags"`
	ScheduleAt string   `json:"schedule_at,omitempty" jsonschema:"RFC3339 schedule time"`
	Visibility string   `json:"visibility,omitempty" jsonschema:"visibility level"`
	Products   []string `json:"products,omitempty" jsonschema:"product keywords"`
}

// SearchFeedsArgs MCP args for search.
type SearchFeedsArgs struct {
	AccountScopedArgs
	Keyword string       `json:"keyword" jsonschema:"search keyword"`
	Filters FilterOption `json:"filters,omitempty" jsonschema:"search filters"`
}

// FilterOption search filters.
type FilterOption struct {
	SortBy      string `json:"sort_by,omitempty" jsonschema:"sort by"`
	NoteType    string `json:"note_type,omitempty" jsonschema:"note type"`
	PublishTime string `json:"publish_time,omitempty" jsonschema:"publish time"`
	SearchScope string `json:"search_scope,omitempty" jsonschema:"search scope"`
	Location    string `json:"location,omitempty" jsonschema:"location"`
}

// FeedDetailArgs MCP args for feed detail.
type FeedDetailArgs struct {
	AccountScopedArgs
	FeedID           string `json:"feed_id" jsonschema:"feed id"`
	XsecToken        string `json:"xsec_token" jsonschema:"feed xsec token"`
	LoadAllComments  bool   `json:"load_all_comments,omitempty" jsonschema:"load all comments"`
	Limit            int    `json:"limit,omitempty" jsonschema:"max parent comments"`
	ClickMoreReplies bool   `json:"click_more_replies,omitempty" jsonschema:"expand replies"`
	ReplyLimit       int    `json:"reply_limit,omitempty" jsonschema:"skip reply thread threshold"`
	ScrollSpeed      string `json:"scroll_speed,omitempty" jsonschema:"slow|normal|fast"`
}

// UserProfileArgs MCP args for user profile.
type UserProfileArgs struct {
	AccountScopedArgs
	UserID    string `json:"user_id" jsonschema:"user id"`
	XsecToken string `json:"xsec_token" jsonschema:"user xsec token"`
}

// PostCommentArgs MCP args for posting comments.
type PostCommentArgs struct {
	AccountScopedArgs
	FeedID    string `json:"feed_id" jsonschema:"feed id"`
	XsecToken string `json:"xsec_token" jsonschema:"feed xsec token"`
	Content   string `json:"content" jsonschema:"comment content"`
}

// ReplyCommentArgs MCP args for replying comments.
type ReplyCommentArgs struct {
	AccountScopedArgs
	FeedID    string `json:"feed_id" jsonschema:"feed id"`
	XsecToken string `json:"xsec_token" jsonschema:"feed xsec token"`
	CommentID string `json:"comment_id,omitempty" jsonschema:"target comment id"`
	UserID    string `json:"user_id,omitempty" jsonschema:"target user id"`
	Content   string `json:"content" jsonschema:"reply content"`
}

// LikeFeedArgs MCP args for like action.
type LikeFeedArgs struct {
	AccountScopedArgs
	FeedID    string `json:"feed_id" jsonschema:"feed id"`
	XsecToken string `json:"xsec_token" jsonschema:"feed xsec token"`
	Unlike    bool   `json:"unlike,omitempty" jsonschema:"true for unlike"`
}

// FavoriteFeedArgs MCP args for favorite action.
type FavoriteFeedArgs struct {
	AccountScopedArgs
	FeedID     string `json:"feed_id" jsonschema:"feed id"`
	XsecToken  string `json:"xsec_token" jsonschema:"feed xsec token"`
	Unfavorite bool   `json:"unfavorite,omitempty" jsonschema:"true for unfavorite"`
}

func InitMCPServer(appServer *AppServer) *mcp.Server {
	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "xiaohongshuritter",
			Version: "3.0.0",
		},
		nil,
	)

	registerTools(server, appServer)
	logrus.Info("MCP Server initialized")
	return server
}

func withPanicRecovery[T any](
	toolName string,
	handler func(context.Context, *mcp.CallToolRequest, T) (*mcp.CallToolResult, any, error),
) func(context.Context, *mcp.CallToolRequest, T) (*mcp.CallToolResult, any, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, args T) (result *mcp.CallToolResult, resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				logrus.WithFields(logrus.Fields{"tool": toolName, "panic": r}).Error("tool handler panicked")
				logrus.Errorf("stack trace:\n%s", debug.Stack())
				result = &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("tool %s panicked: %v", toolName, r)}},
					IsError: true,
				}
				resp = nil
				err = nil
			}
		}()
		return handler(ctx, req, args)
	}
}

func resolveScopedContext(appServer *AppServer, ctx context.Context, scope AccountScope) (context.Context, *mcp.CallToolResult) {
	scopedCtx, _, scopeErr := appServer.resolveScopeForMCP(ctx, scope)
	if scopeErr != nil {
		return ctx, convertToMCPResult(scopeErr)
	}
	return scopedCtx, nil
}

func registerTools(server *mcp.Server, appServer *AppServer) {
	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "check_login_status",
			Description: "check xiaohongshu login status for tenant/account",
			Annotations: &mcp.ToolAnnotations{Title: "Check Login Status", ReadOnlyHint: true},
		},
		withPanicRecovery("check_login_status", func(ctx context.Context, req *mcp.CallToolRequest, args AccountScopedArgs) (*mcp.CallToolResult, any, error) {
			scopedCtx, scopeErr := resolveScopedContext(appServer, ctx, args.Scope())
			if scopeErr != nil {
				return scopeErr, nil, nil
			}
			result := appServer.handleCheckLoginStatus(scopedCtx)
			return convertToMCPResult(result), nil, nil
		}),
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "get_login_qrcode",
			Description: "get login qrcode for tenant/account",
			Annotations: &mcp.ToolAnnotations{Title: "Get Login QR Code", ReadOnlyHint: true},
		},
		withPanicRecovery("get_login_qrcode", func(ctx context.Context, req *mcp.CallToolRequest, args AccountScopedArgs) (*mcp.CallToolResult, any, error) {
			scopedCtx, scopeErr := resolveScopedContext(appServer, ctx, args.Scope())
			if scopeErr != nil {
				return scopeErr, nil, nil
			}
			result := appServer.handleGetLoginQrcode(scopedCtx)
			return convertToMCPResult(result), nil, nil
		}),
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "delete_cookies",
			Description: "delete cookies for tenant/account",
			Annotations: &mcp.ToolAnnotations{Title: "Delete Cookies", DestructiveHint: boolPtr(true)},
		},
		withPanicRecovery("delete_cookies", func(ctx context.Context, req *mcp.CallToolRequest, args AccountScopedArgs) (*mcp.CallToolResult, any, error) {
			scopedCtx, scopeErr := resolveScopedContext(appServer, ctx, args.Scope())
			if scopeErr != nil {
				return scopeErr, nil, nil
			}
			result := appServer.handleDeleteCookies(scopedCtx)
			return convertToMCPResult(result), nil, nil
		}),
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "list_accounts",
			Description: "list configured tenants/accounts and runtime concurrency",
			Annotations: &mcp.ToolAnnotations{Title: "List Accounts", ReadOnlyHint: true},
		},
		withPanicRecovery("list_accounts", func(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
			result := appServer.handleListAccounts(ctx)
			return convertToMCPResult(result), nil, nil
		}),
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "publish_content",
			Description: "publish xiaohongshu image content",
			Annotations: &mcp.ToolAnnotations{Title: "Publish Content", DestructiveHint: boolPtr(true)},
		},
		withPanicRecovery("publish_content", func(ctx context.Context, req *mcp.CallToolRequest, args PublishContentArgs) (*mcp.CallToolResult, any, error) {
			scopedCtx, scopeErr := resolveScopedContext(appServer, ctx, args.Scope())
			if scopeErr != nil {
				return scopeErr, nil, nil
			}
			argsMap := map[string]interface{}{
				"title":       args.Title,
				"content":     args.Content,
				"images":      convertStringsToInterfaces(args.Images),
				"tags":        convertStringsToInterfaces(args.Tags),
				"schedule_at": args.ScheduleAt,
				"is_original": args.IsOriginal,
				"visibility":  args.Visibility,
				"products":    convertStringsToInterfaces(args.Products),
			}
			result := appServer.handlePublishContent(scopedCtx, argsMap)
			return convertToMCPResult(result), nil, nil
		}),
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "list_feeds",
			Description: "list feed recommendations",
			Annotations: &mcp.ToolAnnotations{Title: "List Feeds", ReadOnlyHint: true},
		},
		withPanicRecovery("list_feeds", func(ctx context.Context, req *mcp.CallToolRequest, args AccountScopedArgs) (*mcp.CallToolResult, any, error) {
			scopedCtx, scopeErr := resolveScopedContext(appServer, ctx, args.Scope())
			if scopeErr != nil {
				return scopeErr, nil, nil
			}
			result := appServer.handleListFeeds(scopedCtx)
			return convertToMCPResult(result), nil, nil
		}),
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "search_feeds",
			Description: "search xiaohongshu feeds",
			Annotations: &mcp.ToolAnnotations{Title: "Search Feeds", ReadOnlyHint: true},
		},
		withPanicRecovery("search_feeds", func(ctx context.Context, req *mcp.CallToolRequest, args SearchFeedsArgs) (*mcp.CallToolResult, any, error) {
			scopedCtx, scopeErr := resolveScopedContext(appServer, ctx, args.Scope())
			if scopeErr != nil {
				return scopeErr, nil, nil
			}
			result := appServer.handleSearchFeeds(scopedCtx, args)
			return convertToMCPResult(result), nil, nil
		}),
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "get_feed_detail",
			Description: "get feed detail and optional comments",
			Annotations: &mcp.ToolAnnotations{Title: "Get Feed Detail", ReadOnlyHint: true},
		},
		withPanicRecovery("get_feed_detail", func(ctx context.Context, req *mcp.CallToolRequest, args FeedDetailArgs) (*mcp.CallToolResult, any, error) {
			scopedCtx, scopeErr := resolveScopedContext(appServer, ctx, args.Scope())
			if scopeErr != nil {
				return scopeErr, nil, nil
			}
			argsMap := map[string]interface{}{
				"feed_id":           args.FeedID,
				"xsec_token":        args.XsecToken,
				"load_all_comments": args.LoadAllComments,
			}
			if args.LoadAllComments {
				argsMap["click_more_replies"] = args.ClickMoreReplies
				limit := args.Limit
				if limit <= 0 {
					limit = 20
				}
				argsMap["max_comment_items"] = limit
				replyLimit := args.ReplyLimit
				if replyLimit <= 0 {
					replyLimit = 10
				}
				argsMap["max_replies_threshold"] = replyLimit
				if args.ScrollSpeed != "" {
					argsMap["scroll_speed"] = args.ScrollSpeed
				}
			}
			result := appServer.handleGetFeedDetail(scopedCtx, argsMap)
			return convertToMCPResult(result), nil, nil
		}),
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "user_profile",
			Description: "get user profile by user id",
			Annotations: &mcp.ToolAnnotations{Title: "User Profile", ReadOnlyHint: true},
		},
		withPanicRecovery("user_profile", func(ctx context.Context, req *mcp.CallToolRequest, args UserProfileArgs) (*mcp.CallToolResult, any, error) {
			scopedCtx, scopeErr := resolveScopedContext(appServer, ctx, args.Scope())
			if scopeErr != nil {
				return scopeErr, nil, nil
			}
			argsMap := map[string]interface{}{"user_id": args.UserID, "xsec_token": args.XsecToken}
			result := appServer.handleUserProfile(scopedCtx, argsMap)
			return convertToMCPResult(result), nil, nil
		}),
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "post_comment_to_feed",
			Description: "post comment to feed",
			Annotations: &mcp.ToolAnnotations{Title: "Post Comment", DestructiveHint: boolPtr(true)},
		},
		withPanicRecovery("post_comment_to_feed", func(ctx context.Context, req *mcp.CallToolRequest, args PostCommentArgs) (*mcp.CallToolResult, any, error) {
			scopedCtx, scopeErr := resolveScopedContext(appServer, ctx, args.Scope())
			if scopeErr != nil {
				return scopeErr, nil, nil
			}
			argsMap := map[string]interface{}{"feed_id": args.FeedID, "xsec_token": args.XsecToken, "content": args.Content}
			result := appServer.handlePostComment(scopedCtx, argsMap)
			return convertToMCPResult(result), nil, nil
		}),
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "reply_comment_in_feed",
			Description: "reply to comment in feed",
			Annotations: &mcp.ToolAnnotations{Title: "Reply Comment", DestructiveHint: boolPtr(true)},
		},
		withPanicRecovery("reply_comment_in_feed", func(ctx context.Context, req *mcp.CallToolRequest, args ReplyCommentArgs) (*mcp.CallToolResult, any, error) {
			scopedCtx, scopeErr := resolveScopedContext(appServer, ctx, args.Scope())
			if scopeErr != nil {
				return scopeErr, nil, nil
			}
			if args.CommentID == "" && args.UserID == "" {
				return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: "missing comment_id or user_id"}}}, nil, nil
			}
			argsMap := map[string]interface{}{
				"feed_id":    args.FeedID,
				"xsec_token": args.XsecToken,
				"comment_id": args.CommentID,
				"user_id":    args.UserID,
				"content":    args.Content,
			}
			result := appServer.handleReplyComment(scopedCtx, argsMap)
			return convertToMCPResult(result), nil, nil
		}),
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "publish_with_video",
			Description: "publish xiaohongshu video",
			Annotations: &mcp.ToolAnnotations{Title: "Publish Video", DestructiveHint: boolPtr(true)},
		},
		withPanicRecovery("publish_with_video", func(ctx context.Context, req *mcp.CallToolRequest, args PublishVideoArgs) (*mcp.CallToolResult, any, error) {
			scopedCtx, scopeErr := resolveScopedContext(appServer, ctx, args.Scope())
			if scopeErr != nil {
				return scopeErr, nil, nil
			}
			argsMap := map[string]interface{}{
				"title":       args.Title,
				"content":     args.Content,
				"video":       args.Video,
				"tags":        convertStringsToInterfaces(args.Tags),
				"schedule_at": args.ScheduleAt,
				"visibility":  args.Visibility,
				"products":    convertStringsToInterfaces(args.Products),
			}
			result := appServer.handlePublishVideo(scopedCtx, argsMap)
			return convertToMCPResult(result), nil, nil
		}),
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "like_feed",
			Description: "like or unlike feed",
			Annotations: &mcp.ToolAnnotations{Title: "Like Feed", DestructiveHint: boolPtr(true)},
		},
		withPanicRecovery("like_feed", func(ctx context.Context, req *mcp.CallToolRequest, args LikeFeedArgs) (*mcp.CallToolResult, any, error) {
			scopedCtx, scopeErr := resolveScopedContext(appServer, ctx, args.Scope())
			if scopeErr != nil {
				return scopeErr, nil, nil
			}
			argsMap := map[string]interface{}{"feed_id": args.FeedID, "xsec_token": args.XsecToken, "unlike": args.Unlike}
			result := appServer.handleLikeFeed(scopedCtx, argsMap)
			return convertToMCPResult(result), nil, nil
		}),
	)

	mcp.AddTool(server,
		&mcp.Tool{
			Name:        "favorite_feed",
			Description: "favorite or unfavorite feed",
			Annotations: &mcp.ToolAnnotations{Title: "Favorite Feed", DestructiveHint: boolPtr(true)},
		},
		withPanicRecovery("favorite_feed", func(ctx context.Context, req *mcp.CallToolRequest, args FavoriteFeedArgs) (*mcp.CallToolResult, any, error) {
			scopedCtx, scopeErr := resolveScopedContext(appServer, ctx, args.Scope())
			if scopeErr != nil {
				return scopeErr, nil, nil
			}
			argsMap := map[string]interface{}{"feed_id": args.FeedID, "xsec_token": args.XsecToken, "unfavorite": args.Unfavorite}
			result := appServer.handleFavoriteFeed(scopedCtx, argsMap)
			return convertToMCPResult(result), nil, nil
		}),
	)

	logrus.Infof("Registered %d MCP tools", 14)
}

func convertToMCPResult(result *MCPToolResult) *mcp.CallToolResult {
	var contents []mcp.Content
	for _, c := range result.Content {
		switch c.Type {
		case "text":
			contents = append(contents, &mcp.TextContent{Text: c.Text})
		case "image":
			imageData, err := base64.StdEncoding.DecodeString(c.Data)
			if err != nil {
				logrus.WithError(err).Error("failed to decode base64 image data")
				contents = append(contents, &mcp.TextContent{Text: "image decode failed: " + err.Error()})
			} else {
				contents = append(contents, &mcp.ImageContent{Data: imageData, MIMEType: c.MimeType})
			}
		}
	}

	return &mcp.CallToolResult{Content: contents, IsError: result.IsError}
}

func convertStringsToInterfaces(strs []string) []interface{} {
	result := make([]interface{}, len(strs))
	for i, s := range strs {
		result[i] = s
	}
	return result
}

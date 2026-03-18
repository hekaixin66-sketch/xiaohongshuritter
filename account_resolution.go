package main

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
)

func (s *AppServer) resolveScopeForContext(ctx context.Context, scope AccountScope) (context.Context, AccountScope, error) {
	resolved, err := s.xiaohongshuService.ResolveScope(scope)
	if err != nil {
		return ctx, AccountScope{}, err
	}
	return WithAccountScope(ctx, resolved), resolved, nil
}

func (s *AppServer) resolveScopeForHTTP(c *gin.Context, scope AccountScope) (context.Context, AccountScope, error) {
	requested := AccountScope{
		TenantID: firstNonEmpty(
			scope.TenantID,
			c.GetHeader("X-XHS-Tenant"),
			c.Query("tenant_id"),
			c.Query("tenant"),
		),
		AccountID: firstNonEmpty(
			scope.AccountID,
			c.GetHeader("X-XHS-Account"),
			c.Query("account_id"),
			c.Query("account"),
		),
	}
	return s.resolveScopeForContext(c.Request.Context(), requested)
}

func (s *AppServer) resolveScopeForMCP(ctx context.Context, scope AccountScope) (context.Context, AccountScope, *MCPToolResult) {
	scopedCtx, resolved, err := s.resolveScopeForContext(ctx, scope)
	if err == nil {
		return scopedCtx, resolved, nil
	}
	return ctx, AccountScope{}, &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: fmt.Sprintf("invalid tenant/account selection: %v", err),
		}},
		IsError: true,
	}
}

func scopeLabel(scope AccountScope) string {
	n := scope.Normalized()
	if n.TenantID == "" && n.AccountID == "" {
		return "unknown"
	}
	if n.TenantID == "" {
		return n.AccountID
	}
	if n.AccountID == "" {
		return n.TenantID
	}
	return n.TenantID + "/" + n.AccountID
}

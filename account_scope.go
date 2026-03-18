package main

import (
	"context"
	"strings"
)

type AccountScope struct {
	TenantID  string `json:"tenant_id,omitempty" form:"tenant_id"`
	AccountID string `json:"account_id,omitempty" form:"account_id"`
}

func (s AccountScope) Normalized() AccountScope {
	return AccountScope{
		TenantID:  strings.TrimSpace(s.TenantID),
		AccountID: strings.TrimSpace(s.AccountID),
	}
}

func (s AccountScope) Label() string {
	n := s.Normalized()
	if n.TenantID == "" && n.AccountID == "" {
		return ""
	}
	if n.TenantID == "" {
		return n.AccountID
	}
	if n.AccountID == "" {
		return n.TenantID
	}
	return n.TenantID + "/" + n.AccountID
}

type accountScopeContextKey struct{}

func WithAccountScope(ctx context.Context, scope AccountScope) context.Context {
	return context.WithValue(ctx, accountScopeContextKey{}, scope.Normalized())
}

func AccountScopeFromContext(ctx context.Context) AccountScope {
	if scope, ok := ctx.Value(accountScopeContextKey{}).(AccountScope); ok {
		return scope.Normalized()
	}
	return AccountScope{}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		v := strings.TrimSpace(value)
		if v != "" {
			return v
		}
	}
	return ""
}

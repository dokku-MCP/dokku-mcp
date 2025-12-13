package shared

import (
	"context"
	"time"
)

type TenantContext struct {
	TenantID        string
	UserID          string
	Permissions     []string
	Metadata        map[string]string
	AuthenticatedAt time.Time
	ExpiresAt       *time.Time
}

func (tc *TenantContext) HasPermission(permission string) bool {
	for _, p := range tc.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

func (tc *TenantContext) IsExpired() bool {
	if tc.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*tc.ExpiresAt)
}

func (tc *TenantContext) GetMetadata(key string) (string, bool) {
	value, exists := tc.Metadata[key]
	return value, exists
}

func (tc *TenantContext) SetMetadata(key, value string) {
	if tc.Metadata == nil {
		tc.Metadata = make(map[string]string)
	}
	tc.Metadata[key] = value
}

type contextKey string

const TenantContextKey contextKey = "tenant"

func WithTenantContext(ctx context.Context, tenant *TenantContext) context.Context {
	return context.WithValue(ctx, TenantContextKey, tenant)
}

func GetTenantContext(ctx context.Context) (*TenantContext, bool) {
	tenant, ok := ctx.Value(TenantContextKey).(*TenantContext)
	return tenant, ok
}

func MustGetTenantContext(ctx context.Context) *TenantContext {
	tenant, ok := GetTenantContext(ctx)
	if !ok {
		panic("tenant context not found in context")
	}
	return tenant
}

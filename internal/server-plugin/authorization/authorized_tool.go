package authorization

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/server/auth"
	"github.com/dokku-mcp/dokku-mcp/internal/shared"
	"github.com/mark3labs/mcp-go/mcp"
)

func WrapToolWithAuthorization(
	tool domain.Tool,
	resource string,
	action string,
	authChecker auth.AuthorizationChecker,
	logger *slog.Logger,
) domain.Tool {
	originalHandler := tool.Handler
	authorizedHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tenantCtx, hasTenant := shared.GetTenantContext(ctx)

		if hasTenant && authChecker != nil {
			logger.Debug("Checking authorization",
				"tool", tool.Name,
				"tenant_id", tenantCtx.TenantID,
				"user_id", tenantCtx.UserID,
				"resource", resource,
				"action", action)

			if err := authChecker.CheckPermission(ctx, tenantCtx, resource, action); err != nil {
				logger.Warn("Authorization failed",
					"tool", tool.Name,
					"tenant_id", tenantCtx.TenantID,
					"user_id", tenantCtx.UserID,
					"resource", resource,
					"action", action,
					"error", err)

				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf("Permission denied: %v", err),
						},
					},
				}, fmt.Errorf("permission denied: %w", err)
			}

			logger.Debug("Authorization successful",
				"tool", tool.Name,
				"tenant_id", tenantCtx.TenantID,
				"user_id", tenantCtx.UserID)
		}

		return originalHandler(ctx, request)
	}

	return domain.Tool{
		Name:        tool.Name,
		Description: tool.Description,
		Builder:     tool.Builder,
		Handler:     authorizedHandler,
	}
}

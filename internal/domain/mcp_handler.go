package domain

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// MCPHandler defines the interface for handling MCP operations
type MCPHandler interface {
	// Resource handlers
	HandleApplicationsResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error)

	// Tool handlers
	HandleCreateApplication(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)
	HandleDeployApplication(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)
	HandleScaleApplication(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)
	HandleSetApplicationConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

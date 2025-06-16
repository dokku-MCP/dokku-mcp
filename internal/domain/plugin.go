package domain

import (
	"context"

	"github.com/mark3labs/mcp-go/server"
)

// FeaturePlugin represents a self-contained feature that can be dynamically activated.
// Each plugin encapsulates its own resources, tools, and prompts, and knows which
// Dokku plugin it depends on for activation.
type FeaturePlugin interface {
	Name() string

	// Return an empty string for core plugins that are always active.
	DokkuPluginName() string

	// Registers the plugin's resources, tools, and prompts.
	Register(mcpServer *server.MCPServer) error

	// Deregisters the plugin's resources, tools, and prompts from the MCP server.
	Deregister(mcpServer *server.MCPServer) error
}

// Abstracts the infrastructure concerns of checking plugin availability.
type PluginDiscoveryService interface {
	// Returns an empty slice if no plugins are enabled or if discovery fails.
	GetEnabledPlugins(ctx context.Context) ([]string, error)
}

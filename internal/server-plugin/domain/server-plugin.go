package domain

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ServerPlugin represents the unified plugin interface
// Each plugin only needs to provide its basic information and capabilities
type ServerPlugin interface {
	ID() string
	Name() string
	Description() string
	Version() string

	// Optional dependency on Dokku plugin (empty string means no dependency)
	DokkuPluginName() string
}

// ResourceProvider defines plugins that can provide resources
type ResourceProvider interface {
	ServerPlugin
	GetResources(ctx context.Context) ([]Resource, error)
}

// ToolProvider defines plugins that can provide tools
type ToolProvider interface {
	ServerPlugin
	GetTools(ctx context.Context) ([]Tool, error)
}

// PromptProvider defines plugins that can provide prompts
type PromptProvider interface {
	ServerPlugin
	GetPrompts(ctx context.Context) ([]Prompt, error)
}

// Resource represents a plugin resource capability
type Resource struct {
	URI         string
	Name        string
	Description string
	MIMEType    string
	Handler     ResourceHandler
}

// Tool represents a plugin tool capability
type Tool struct {
	Name        string
	Description string
	Builder     func() mcp.Tool
	Handler     ToolHandler
}

// Prompt represents a plugin prompt capability
type Prompt struct {
	Name        string
	Description string
	Builder     func() mcp.Prompt
	Handler     PromptHandler
}

// PromptArgument represents a prompt argument
type PromptArgument struct {
	Name        string
	Description string
	Required    bool
}

// Handler type aliases - properly reference MCP server types
type ResourceHandler = server.ResourceHandlerFunc
type ToolHandler = server.ToolHandlerFunc
type PromptHandler = server.PromptHandlerFunc

// ServerPluginDiscoveryService defines the interface for discovering enabled Dokku plugins
type ServerPluginDiscoveryService interface {
	GetEnabledDokkuPlugins(ctx context.Context) ([]string, error)
}

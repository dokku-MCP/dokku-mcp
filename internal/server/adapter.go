package server

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ServerPluginProvider interface defines what we need from the plugin registry
type ServerPluginProvider interface {
	GetResourceProviders() []domain.ResourceProvider
	GetToolProviders() []domain.ToolProvider
	GetPromptProviders() []domain.PromptProvider
}

// DynamicServerPluginProvider provides access to only active plugins
type DynamicServerPluginProvider interface {
	GetActiveServerPlugins() []domain.ServerPlugin
}

// MCPAdapter bridges between our plugin system and the MCP server
// Single responsibility: Adapt plugin capabilities to MCP server registration
type MCPAdapter struct {
	dynamicRegistry DynamicServerPluginProvider
	mcpServer       *server.MCPServer
	logger          *slog.Logger
}

// NewMCPAdapter creates a new MCP adapter using the dynamic registry
func NewMCPAdapter(dynamicRegistry DynamicServerPluginProvider, mcpServer *server.MCPServer, logger *slog.Logger) *MCPAdapter {
	return &MCPAdapter{
		dynamicRegistry: dynamicRegistry,
		mcpServer:       mcpServer,
		logger:          logger,
	}
}

// GetResourceProviders returns resource providers from active plugins only
func (a *MCPAdapter) GetResourceProviders() []domain.ResourceProvider {
	var providers []domain.ResourceProvider
	for _, plugin := range a.dynamicRegistry.GetActiveServerPlugins() {
		if provider, ok := plugin.(domain.ResourceProvider); ok {
			providers = append(providers, provider)
		}
	}
	return providers
}

// GetToolProviders returns tool providers from active plugins only
func (a *MCPAdapter) GetToolProviders() []domain.ToolProvider {
	var providers []domain.ToolProvider
	for _, plugin := range a.dynamicRegistry.GetActiveServerPlugins() {
		if provider, ok := plugin.(domain.ToolProvider); ok {
			providers = append(providers, provider)
		}
	}
	return providers
}

// GetPromptProviders returns prompt providers from active plugins only
func (a *MCPAdapter) GetPromptProviders() []domain.PromptProvider {
	var providers []domain.PromptProvider
	for _, plugin := range a.dynamicRegistry.GetActiveServerPlugins() {
		if provider, ok := plugin.(domain.PromptProvider); ok {
			providers = append(providers, provider)
		}
	}
	return providers
}

// RegisterAllServerPlugins registers all plugins from the registry with the MCP server
func (a *MCPAdapter) RegisterAllServerPlugins(ctx context.Context) error {
	a.logger.Info("Registering all plugins with MCP server")

	if err := a.registerResources(ctx); err != nil {
		return fmt.Errorf("failed to register resources: %w", err)
	}

	if err := a.registerTools(ctx); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}

	if err := a.registerPrompts(ctx); err != nil {
		return fmt.Errorf("failed to register prompts: %w", err)
	}

	a.logger.Info("All plugins registered successfully")
	return nil
}

// registerResources registers all resources from resource providers
func (a *MCPAdapter) registerResources(ctx context.Context) error {
	providers := a.GetResourceProviders()
	a.logger.Debug("Starting resource registration", "provider_count", len(providers))

	for _, provider := range providers {
		a.logger.Debug("Processing resource provider", "plugin_id", provider.ID())

		resources, err := provider.GetResources(ctx)
		if err != nil {
			a.logger.Error("Failed to get resources from provider",
				"plugin", provider.ID(), "error", err)
			continue
		}

		a.logger.Debug("Got resources from provider",
			"plugin_id", provider.ID(),
			"resource_count", len(resources))

		for _, resource := range resources {
			mcpResource := mcp.NewResource(
				resource.URI,
				resource.Name,
				mcp.WithResourceDescription(resource.Description),
				mcp.WithMIMEType(resource.MIMEType),
			)

			a.mcpServer.AddResource(mcpResource, resource.Handler)
			a.logger.Debug("Resource registered",
				"plugin", provider.ID(),
				"resource", resource.Name,
				"uri", resource.URI)
		}
	}

	a.logger.Debug("Resource registration completed")
	return nil
}

// registerTools registers all tools from tool providers
func (a *MCPAdapter) registerTools(ctx context.Context) error {
	providers := a.GetToolProviders()
	a.logger.Debug("Starting tool registration", "provider_count", len(providers))

	for _, provider := range providers {
		a.logger.Debug("Processing tool provider", "plugin_id", provider.ID())

		tools, err := provider.GetTools(ctx)
		if err != nil {
			a.logger.Error("Failed to get tools from provider",
				"plugin", provider.ID(), "error", err)
			continue
		}

		a.logger.Debug("Got tools from provider",
			"plugin_id", provider.ID(),
			"tool_count", len(tools))

		for _, tool := range tools {
			// Use the builder pattern to create the MCP tool
			mcpTool := tool.Builder()

			a.mcpServer.AddTool(mcpTool, tool.Handler)
			a.logger.Debug("Tool registered",
				"plugin", provider.ID(),
				"tool", tool.Name)
		}
	}

	a.logger.Debug("Tool registration completed")
	return nil
}

// registerPrompts registers all prompts from prompt providers
func (a *MCPAdapter) registerPrompts(ctx context.Context) error {
	providers := a.GetPromptProviders()

	for _, provider := range providers {
		prompts, err := provider.GetPrompts(ctx)
		if err != nil {
			a.logger.Error("Failed to get prompts from provider",
				"plugin", provider.ID(), "error", err)
			continue
		}

		for _, prompt := range prompts {
			// Use the builder pattern to create the MCP prompt
			mcpPrompt := prompt.Builder()

			a.mcpServer.AddPrompt(mcpPrompt, prompt.Handler)
			a.logger.Debug("Prompt registered",
				"plugin", provider.ID(),
				"prompt", prompt.Name)
		}
	}

	return nil
}

// RegisterServerPlugin registers a single server plugin with the MCP server
func (a *MCPAdapter) RegisterServerPlugin(ctx context.Context, plugin domain.ServerPlugin) error {
	// Register resources if server plugin provides them
	if resourceProvider, ok := plugin.(domain.ResourceProvider); ok {
		resources, err := resourceProvider.GetResources(ctx)
		if err == nil {
			for _, resource := range resources {
				mcpResource := mcp.NewResource(
					resource.URI,
					resource.Name,
					mcp.WithResourceDescription(resource.Description),
					mcp.WithMIMEType(resource.MIMEType),
				)
				a.mcpServer.AddResource(mcpResource, resource.Handler)
			}
		}
	}

	// Register tools if server plugin provides them
	if toolProvider, ok := plugin.(domain.ToolProvider); ok {
		tools, err := toolProvider.GetTools(ctx)
		if err == nil {
			for _, tool := range tools {
				mcpTool := tool.Builder()
				a.mcpServer.AddTool(mcpTool, tool.Handler)
			}
		}
	}

	// Register prompts if server plugin provides them
	if promptProvider, ok := plugin.(domain.PromptProvider); ok {
		prompts, err := promptProvider.GetPrompts(ctx)
		if err == nil {
			for _, prompt := range prompts {
				mcpPrompt := prompt.Builder()
				a.mcpServer.AddPrompt(mcpPrompt, prompt.Handler)
			}
		}
	}

	a.logger.Debug("ServerPlugin registered with MCP server", "server-plugin", plugin.ID())
	return nil
}

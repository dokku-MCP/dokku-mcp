package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	dokkuApi "github.com/alex-galey/dokku-mcp/internal/dokku-api"
	serverDomain "github.com/alex-galey/dokku-mcp/internal/server-plugin/domain"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/core/application"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/core/infrastructure"
	"github.com/mark3labs/mcp-go/mcp"
)

// CoreServerPlugin provides core Dokku functionality and global configuration
type CoreServerPlugin struct {
	coreService *application.CoreService
	logger      *slog.Logger
}

// NewCoreServerPlugin creates a new core functionality server plugin
func NewCoreServerPlugin(client dokkuApi.DokkuClient, logger *slog.Logger) serverDomain.ServerPlugin {
	// Create infrastructure adapter
	adapter := infrastructure.NewDokkuCoreAdapter(client, logger)

	// Create application service
	coreService := application.NewCoreService(
		adapter, // SystemRepository
		adapter, // PluginRepository
		adapter, // DomainRepository
		adapter, // SSHKeyRepository
		adapter, // RegistryRepository
		adapter, // ConfigurationRepository
		logger,
	)

	return &CoreServerPlugin{
		coreService: coreService,
		logger:      logger,
	}
}

// ServerPlugin interface implementation
func (p *CoreServerPlugin) ID() string {
	return "core"
}

func (p *CoreServerPlugin) Name() string {
	return "Core Functionality"
}

func (p *CoreServerPlugin) Description() string {
	return "Core Dokku functionality including system status, global configuration, plugin management, domain management, SSH keys, and Docker registry management"
}

func (p *CoreServerPlugin) Version() string {
	return "0.1.0"
}

func (p *CoreServerPlugin) DokkuPluginName() string {
	return "" // Core plugin - no Dokku dependency
}

// ResourceProvider implementation
func (p *CoreServerPlugin) GetResources(ctx context.Context) ([]serverDomain.Resource, error) {
	p.logger.Debug("Core plugin: Getting MCP resources")

	resources := []serverDomain.Resource{
		// System Status Resource
		{
			URI:         "dokku://core/system/status",
			Name:        "System Status",
			Description: "Current Dokku server status including version, global configuration, and resource usage",
			MIMEType:    "application/json",
			Handler:     p.handleSystemStatusResource,
		},

		// Server Info Resource (comprehensive)
		{
			URI:         "dokku://core/server/info",
			Name:        "Server Information",
			Description: "Complete server information including system status, plugins, domains, SSH keys, and configuration",
			MIMEType:    "application/json",
			Handler:     p.handleServerInfoResource,
		},

		// Plugin List Resource
		{
			URI:         "dokku://core/plugins",
			Name:        "Dokku Plugins",
			Description: "List of all installed Dokku plugins with their status and versions",
			MIMEType:    "application/json",
			Handler:     p.handlePluginsResource,
		},
	}

	p.logger.Debug("Core plugin: Generated resources", "count", len(resources))
	return resources, nil
}

// Resource handler methods
func (p *CoreServerPlugin) handleSystemStatusResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	status, err := p.coreService.GetSystemStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get system status: %w", err)
	}

	jsonData, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize system status: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (p *CoreServerPlugin) handleServerInfoResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	info, err := p.coreService.GetServerInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}

	jsonData, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize server info: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (p *CoreServerPlugin) handlePluginsResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	plugins, err := p.coreService.ListPlugins(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list plugins: %w", err)
	}

	jsonData, err := json.MarshalIndent(plugins, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize plugins: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// ToolProvider implementation
func (p *CoreServerPlugin) GetTools(ctx context.Context) ([]serverDomain.Tool, error) {
	p.logger.Debug("Core plugin: Getting MCP tools")

	tools := []serverDomain.Tool{
		{
			Name:        "get_system_status",
			Description: "Get the current Dokku system status including version, configuration, and resource usage",
			Builder:     p.buildGetSystemStatusTool,
			Handler:     p.handleGetSystemStatusTool,
		},
		{
			Name:        "list_plugins",
			Description: "List all installed Dokku plugins with their status and versions",
			Builder:     p.buildListPluginsTool,
			Handler:     p.handleListPluginsTool,
		},
	}

	p.logger.Debug("Core plugin: Generated tools", "count", len(tools))
	return tools, nil
}

// Tool builders
func (p *CoreServerPlugin) buildGetSystemStatusTool() mcp.Tool {
	return mcp.NewTool(
		"get_system_status",
		mcp.WithDescription("Get the current Dokku system status including version, configuration, and resource usage"),
	)
}

func (p *CoreServerPlugin) buildListPluginsTool() mcp.Tool {
	return mcp.NewTool(
		"list_plugins",
		mcp.WithDescription("List all installed Dokku plugins with their status and versions"),
	)
}

// Tool handlers
func (p *CoreServerPlugin) handleGetSystemStatusTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	status, err := p.coreService.GetSystemStatus(ctx)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to get system status: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	jsonData, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to serialize system status: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(jsonData),
			},
		},
		IsError: false,
	}, nil
}

func (p *CoreServerPlugin) handleListPluginsTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	plugins, err := p.coreService.ListPlugins(ctx)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to list plugins: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	jsonData, err := json.MarshalIndent(plugins, "", "  ")
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Failed to serialize plugins: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: string(jsonData),
			},
		},
		IsError: false,
	}, nil
}

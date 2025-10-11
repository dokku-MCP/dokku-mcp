package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	dokkuApi "github.com/dokku-mcp/dokku-mcp/internal/dokku-api"
	serverDomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/core/application"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/core/infrastructure"
	"github.com/dokku-mcp/dokku-mcp/pkg/config"
	"github.com/dokku-mcp/dokku-mcp/pkg/logger"
	"github.com/mark3labs/mcp-go/mcp"
)

// CoreServerPlugin provides core Dokku functionality and global configuration
type CoreServerPlugin struct {
	coreService *application.CoreService
	logger      *slog.Logger
	cfg         *config.ServerConfig
}

// NewCoreServerPlugin creates a new core functionality server plugin
func NewCoreServerPlugin(client dokkuApi.DokkuClient, logger *slog.Logger, cfg *config.ServerConfig) serverDomain.ServerPlugin {
	// Create infrastructure adapter
	adapter := infrastructure.NewDokkuCoreAdapter(client, logger)

	// Create application service
	coreService := application.NewCoreService(
		adapter, // SystemRepository
		adapter, // PluginRepository
		adapter, // SSHKeyRepository
		adapter, // RegistryRepository
		adapter, // ConfigurationRepository
		logger,
	)

	return &CoreServerPlugin{
		coreService: coreService,
		logger:      logger,
		cfg:         cfg,
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
	return "Core Dokku functionality including system status, global configuration, plugin management, SSH keys, and Docker registry management"
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

	tools := []serverDomain.Tool{}
	if p.cfg != nil && p.cfg.ExposeServerLogs {
		tools = append(tools, serverDomain.Tool{
			Name:        "get_server_logs",
			Description: "Get recent dokku-mcp server logs (in-memory ring buffer)",
			Builder:     p.buildGetServerLogsTool,
			Handler:     p.handleGetServerLogsTool,
		})
	}

	p.logger.Debug("Core plugin: Generated tools", "count", len(tools))
	return tools, nil
}

// Tool builders
// no builders for system status or plugin list tools; they are resources only

func (p *CoreServerPlugin) buildGetServerLogsTool() mcp.Tool {
	return mcp.NewTool(
		"get_server_logs",
		mcp.WithDescription("Get recent dokku-mcp server logs"),
		mcp.WithNumber("last",
			mcp.Description("Number of last lines to return (default 200)"),
		),
		mcp.WithString("level",
			mcp.Description("Optional filter: debug|info|warn|error"),
		),
		mcp.WithString("contains",
			mcp.Description("Optional substring filter"),
		),
	)
}

// Tool handlers
// no handlers for system status or plugin list tools; they are resources only

func (p *CoreServerPlugin) handleGetServerLogsTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract arguments
	last := 200
	if v, ok := req.GetArguments()["last"]; ok {
		switch n := v.(type) {
		case float64:
			last = int(n)
		case int:
			last = n
		}
		if last <= 0 {
			last = 200
		}
	}
	levelFilter := ""
	if v, ok := req.GetArguments()["level"].(string); ok {
		levelFilter = strings.ToLower(v)
	}
	contains := ""
	if v, ok := req.GetArguments()["contains"].(string); ok {
		contains = v
	}

	// Read from global ring buffer
	lines := logger.GetLogBuffer().GetLast(last)
	// Apply filters
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if levelFilter != "" && !strings.Contains(line, levelFilter) {
			continue
		}
		if contains != "" && !strings.Contains(line, contains) {
			continue
		}
		out = append(out, line)
	}

	// Return as single text block
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{Type: "text", Text: strings.Join(out, "\n")},
		},
	}, nil
}

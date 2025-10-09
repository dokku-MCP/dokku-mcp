package server

import (
	"log/slog"

	dokkuApi "github.com/alex-galey/dokku-mcp/internal/dokku-api"
	plugins "github.com/alex-galey/dokku-mcp/internal/server-plugin/application"
	"github.com/alex-galey/dokku-mcp/internal/server-plugin/domain"
	"github.com/alex-galey/dokku-mcp/internal/server-plugin/infrastructure"
	"github.com/alex-galey/dokku-mcp/pkg/config"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/fx"
)

// NewMCPServerInstance creates a new MCP server instance.
func NewMCPServerInstance(cfg *config.ServerConfig, logger *slog.Logger) *server.MCPServer {
	logger.Debug("Creating MCP server instance")
	version := "dev"
	mcpServer := server.NewMCPServer(
		"Dokku MCP Server",
		version,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
	)
	logger.Debug("MCP server instance created successfully")
	return mcpServer
}

var Module = fx.Module("server",
	fx.Provide(
		NewMCPServerInstance,
		fx.Annotate(
			dokkuApi.NewDokkuClientFromConfig,
			fx.As(new(dokkuApi.DokkuClient)),
		),
		plugins.NewServerPluginRegistry,
		fx.Annotate(
			func(dynamicRegistry *plugins.DynamicServerPluginRegistry, mcpServer *server.MCPServer, logger *slog.Logger) *MCPAdapter {
				return NewMCPAdapter(dynamicRegistry, mcpServer, logger)
			},
		),
		fx.Annotate(
			func(adapter *MCPAdapter) ServerPluginProvider { return adapter },
			fx.As(new(ServerPluginProvider)),
		),
		fx.Annotate(
			infrastructure.NewPluginDiscoveryService,
			fx.As(new(domain.ServerPluginDiscoveryService)),
		),
		plugins.NewDynamicServerPluginRegistry,
	),
	fx.Invoke(registerServerHooks),
	fx.Invoke(func(registry *plugins.DynamicServerPluginRegistry, lc fx.Lifecycle) {
		registry.RegisterHooks(lc)
	}),
)

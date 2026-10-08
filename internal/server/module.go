package server

import (
	"log/slog"

	dokkuApi "github.com/dokku-mcp/dokku-mcp/internal/dokku-api"
	plugins "github.com/dokku-mcp/dokku-mcp/internal/server-plugin/application"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/infrastructure"
	"github.com/dokku-mcp/dokku-mcp/pkg/buildinfo"
	"github.com/dokku-mcp/dokku-mcp/pkg/config"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/fx"
)

// serverInstructions is sent to clients at initialization to orient the model.
const serverInstructions = `This server manages a Dokku PaaS host over SSH.
Start with the dokku://onboarding/quickstart resource to learn the available capabilities.
Deployments run asynchronously: deploy_app returns a deployment ID that you can poll with get_deployment_status.
Destructive operations may be blocked by the server's security policy; report such errors to the user instead of retrying.`

// NewMCPServerInstance creates a new MCP server instance.
func NewMCPServerInstance(cfg *config.ServerConfig, logger *slog.Logger) *server.MCPServer {
	logger.Debug("Creating MCP server instance")
	mcpServer := server.NewMCPServer(
		"Dokku MCP Server",
		buildinfo.Version,
		server.WithTitle("Dokku"),
		server.WithDescription("Manage Dokku applications, deployments, domains and plugins"),
		server.WithWebsiteURL("https://github.com/dokku-mcp/dokku-mcp"),
		server.WithInstructions(serverInstructions),
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
		server.WithRecovery(),
		server.WithResourceRecovery(),
		server.WithInputSchemaValidation(),
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

package main

import (
	"log/slog"

	"github.com/alex-galey/dokku-mcp/internal/application"
	"github.com/alex-galey/dokku-mcp/internal/application/plugins"
	"github.com/alex-galey/dokku-mcp/internal/domain"
	"github.com/alex-galey/dokku-mcp/internal/infrastructure"
	"github.com/alex-galey/dokku-mcp/internal/infrastructure/config"
	"github.com/alex-galey/dokku-mcp/internal/infrastructure/server"
	mcpServer "github.com/mark3labs/mcp-go/server"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		// Layer providers
		infrastructure.InfrastructureProviders,
		domain.DomainProviders,
		application.ApplicationProviders,

		// Lifecycle management - wire concrete implementation to interface
		fx.Invoke(func(registry *plugins.DynamicPluginRegistry, lc fx.Lifecycle) {
			registry.RegisterHooks(lc)
		}),

		// Start server - provide concrete implementation as interface
		fx.Invoke(func(
			lc fx.Lifecycle,
			cfg *config.ServerConfig,
			mcpSrv *mcpServer.MCPServer,
			registry *plugins.DynamicPluginRegistry,
			logger *slog.Logger,
		) {
			// Cast to interface to avoid layer violation
			server.Run(lc, cfg, mcpSrv, registry, logger)
		}),
	).Run()
}

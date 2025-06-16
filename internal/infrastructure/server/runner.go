package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/alex-galey/dokku-mcp/internal/domain"
	"github.com/alex-galey/dokku-mcp/internal/infrastructure/config"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/fx"
)

// PluginRegistry defines the interface for plugin management
// This allows us to avoid importing from application layer
type PluginRegistry interface {
	SyncPlugins(ctx context.Context) error
	GetActivePlugins() []domain.FeaturePlugin
}

// Run sets up the MCP server lifecycle hooks.
// This function is designed to be invoked by Fx.
func Run(
	lc fx.Lifecycle,
	cfg *config.ServerConfig,
	mcpServer *server.MCPServer,
	pluginRegistry PluginRegistry,
	logger *slog.Logger,
) {
	lc.Append(fx.Hook{
		OnStart: func(startCtx context.Context) error {
			logger.Info("Starting Dokku MCP Server",
				"host", cfg.Host,
				"port", cfg.Port)

			// Start the MCP server first
			logger.Info("MCP Server listening on stdio")
			go func() {
				if err := server.ServeStdio(mcpServer); err != nil {
					logger.Error("Server error", "error", err)
				}
			}()

			// Initialize plugins in background to avoid blocking startup
			go func() {
				logger.Info("Initializing plugins in background...")

				// Create a context with longer timeout for plugin initialization
				pluginCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				if err := pluginRegistry.SyncPlugins(pluginCtx); err != nil {
					logger.Error("Failed to initialize plugins", "error", err)
					return
				}

				activePlugins := pluginRegistry.GetActivePlugins()
				logger.Info("Plugins initialized successfully",
					"active_plugins", len(activePlugins))

				for _, plugin := range activePlugins {
					logger.Debug("Active plugin",
						"name", plugin.Name(),
						"dokku_plugin", plugin.DokkuPluginName())
				}
			}()

			return nil
		},
		OnStop: func(context.Context) error {
			logger.Info("Stopping Dokku MCP Server...")
			return nil
		},
	})
}

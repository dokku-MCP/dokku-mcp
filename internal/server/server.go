package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	plugins "github.com/alex-galey/dokku-mcp/internal/server-plugin/application"
	"github.com/alex-galey/dokku-mcp/pkg/config"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/fx"
)

// registerServerHooks uses fx.Hook to manage the server's lifecycle.
func registerServerHooks(lc fx.Lifecycle, cfg *config.ServerConfig, mcpServer *server.MCPServer, adapter *MCPAdapter, dynamicRegistry *plugins.DynamicServerPluginRegistry, logger *slog.Logger) {
	var httpServer *http.Server

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Performing initial plugin synchronization...")

			if err := dynamicRegistry.SyncServerPlugins(ctx); err != nil {
				logger.Error("Initial plugin sync failed", "error", err)
			}

			logger.Info("Registering all server plugins...")
			if err := adapter.RegisterAllServerPlugins(ctx); err != nil {
				return fmt.Errorf("failed to register server plugins: %w", err)
			}
			logger.Info("All plugins registered.")

			switch cfg.Transport.Type {
			case "sse":
				logger.Info("Starting MCP server with 'sse' transport.")
				sseServer := server.NewSSEServer(mcpServer)
				go func() {
					addr := fmt.Sprintf("%s:%d", cfg.Transport.Host, cfg.Transport.Port)
					logger.Info("SSE server listening", "address", addr)
					if err := sseServer.Start(addr); err != nil && err != http.ErrServerClosed {
						logger.Error("SSE server failed", "error", err)
					}
				}()
			case "stdio":
				logger.Info("Starting MCP server with 'stdio' transport.")
				go func() {
					if err := server.ServeStdio(mcpServer); err != nil {
						logger.Error("Stdio server failed", "error", err)
					}
				}()
			default:
				return fmt.Errorf("unknown transport type: %s", cfg.Transport.Type)
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if httpServer != nil {
				logger.Info("Shutting down SSE server gracefully...")
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				return httpServer.Shutdown(shutdownCtx)
			}
			logger.Info("Stdio server shutdown.")
			return nil
		},
	})
}

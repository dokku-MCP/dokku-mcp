package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	plugins "github.com/dokku-mcp/dokku-mcp/internal/server-plugin/application"
	"github.com/dokku-mcp/dokku-mcp/internal/server/auth"
	"github.com/dokku-mcp/dokku-mcp/internal/shared"
	"github.com/dokku-mcp/dokku-mcp/pkg/config"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/fx"
)

type AuthenticatorParams struct {
	fx.In

	Authenticator auth.Authenticator `optional:"true"`
}

func registerServerHooks(
	lc fx.Lifecycle,
	cfg *config.ServerConfig,
	mcpServer *server.MCPServer,
	adapter *MCPAdapter,
	dynamicRegistry *plugins.DynamicServerPluginRegistry,
	authParams AuthenticatorParams,
	logger *slog.Logger,
) {
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

			switch cfg.Transport.Type {
			case "sse":
				addr := fmt.Sprintf("%s:%d", cfg.Transport.Host, cfg.Transport.Port)
				httpServer = &http.Server{
					Addr:              addr,
					ReadHeaderTimeout: 10 * time.Second,
					WriteTimeout:      30 * time.Second,
					IdleTimeout:       120 * time.Second,
					MaxHeaderBytes:    1 << 20, // 1 MB
				}

				var sseServer *server.SSEServer
				var handler http.Handler

				if cfg.MultiTenant.Enabled && authParams.Authenticator != nil {
					logger.Info("Starting MCP server with SSE transport and multi-tenant authentication")

					sseServer = server.NewSSEServer(
						mcpServer,
						server.WithHTTPServer(httpServer),
						server.WithSSEContextFunc(func(ctx context.Context, r *http.Request) context.Context {
							return injectTenantContext(ctx, r, authParams.Authenticator, logger)
						}),
					)
				} else {
					logger.Info("Starting MCP server with SSE transport (single-tenant mode)")
					sseServer = server.NewSSEServer(
						mcpServer,
						server.WithHTTPServer(httpServer),
					)
				}

				// Apply CORS middleware if enabled
				if cfg.Transport.CORS.Enabled {
					logger.Info("CORS middleware enabled",
						"allowed_origins", cfg.Transport.CORS.AllowedOrigins,
						"allowed_methods", cfg.Transport.CORS.AllowedMethods)
					handler = CORSMiddleware(&cfg.Transport.CORS)(sseServer)
					httpServer.Handler = handler
				} else {
					logger.Debug("CORS middleware disabled, using mcp-go default CORS (*)")
				}

				go func() {
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

func injectTenantContext(ctx context.Context, r *http.Request, authenticator auth.Authenticator, logger *slog.Logger) context.Context {
	token := r.URL.Query().Get("token")
	if token == "" {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		logger.Debug("No authentication token provided, using single-tenant mode")
		return ctx
	}

	tenantCtx, err := authenticator.Authenticate(ctx, token)
	if err != nil {
		logger.Warn("Authentication failed",
			"error", err,
			"remote_addr", r.RemoteAddr)
		return ctx
	}

	logger.Debug("Authentication successful",
		"tenant_id", tenantCtx.TenantID,
		"user_id", tenantCtx.UserID,
		"permissions", tenantCtx.Permissions)

	return shared.WithTenantContext(ctx, tenantCtx)
}

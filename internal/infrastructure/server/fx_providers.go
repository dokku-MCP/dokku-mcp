package server

import (
	"log/slog"
	"os"

	"github.com/alex-galey/dokku-mcp/internal/infrastructure/config"
	"github.com/mark3labs/mcp-go/server"
)

// NewSlogLogger creates a structured logger for the application.
func NewSlogLogger(cfg *config.ServerConfig) *slog.Logger {
	var handler slog.Handler

	// Configure log level
	var level slog.Level
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Create handler based on format preference
	opts := &slog.HandlerOptions{
		Level: level,
	}

	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts)
	}

	return slog.New(handler)
}

// NewMCPServerInstance creates a new MCP server instance.
func NewMCPServerInstance(cfg *config.ServerConfig, logger *slog.Logger) *server.MCPServer {
	logger.Debug("Creating MCP server instance")

	// Use a default version - this could be injected via build flags
	version := "dev"

	mcpServer := server.NewMCPServer(
		"Dokku MCP Server",
		version,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(false, true),
		server.WithPromptCapabilities(true),
	)

	logger.Debug("MCP server instance created successfully")
	return mcpServer
}

package logger

import (
	"log/slog"
	"os"

	"github.com/alex-galey/dokku-mcp/pkg/config"
	"go.uber.org/fx"
)

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

var Module = fx.Module("logger",
	fx.Provide(NewSlogLogger),
)

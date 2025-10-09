package logger

import (
	"log/slog"
	"os"

	"github.com/alex-galey/dokku-mcp/pkg/config"
	"go.uber.org/fx"
)

var globalRing = NewRingBuffer(DefaultLogBufferCapacity)

const DefaultLogBufferCapacity = 2000

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

	capacity := DefaultLogBufferCapacity
	if cfg.ExposeServerLogs && cfg.LogBufferCapacity > 0 {
		capacity = cfg.LogBufferCapacity
	}
	globalRing = NewRingBuffer(capacity)
	buffered := newBufferingHandler(handler, globalRing, opts)
	return slog.New(buffered)
}

var Module = fx.Module("logger",
	fx.Provide(NewSlogLogger),
)

// Expose accessors for MCP tools
func GetLogBuffer() *RingBuffer { return globalRing }

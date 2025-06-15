package cli

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/alex-galey/dokku-mcp/src/infrastructure/config"
	mcpserver "github.com/alex-galey/dokku-mcp/src/infrastructure/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type CommandConfig struct {
	Version   string
	BuildTime string
}

func CreateRootCommand(cmdConfig *CommandConfig) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "dokku-mcp",
		Short: "Dokku MCP Server - MCP Server for Dokku",
		Long: `The Dokku MCP Server provides a Model Context Protocol interface to manage
Dokku applications, services and infrastructure through LLM clients like Claude.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(cmdConfig)
		},
	}

	addPersistentFlags(rootCmd)

	if err := bindFlags(rootCmd); err != nil {
		log.Fatalf("Failed to bind flags to configuration: %v", err)
	}

	return rootCmd
}

func CreateVersionCommand(cmdConfig *CommandConfig) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(os.Stderr, "Dokku MCP Server\n")
			fmt.Fprintf(os.Stderr, "Version: %s\n", cmdConfig.Version)
			fmt.Fprintf(os.Stderr, "Build time: %s\n", cmdConfig.BuildTime)
		},
	}
}

func Execute(rootCmd *cobra.Command) {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Failed to execute the command: %v", err)
	}
}

func runServer(cmdConfig *CommandConfig) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load the configuration: %w", err)
	}

	logger := initSlogLogger(cfg)

	serverLogger := logger.With(
		slog.String("version", cmdConfig.Version),
		slog.String("build_time", cmdConfig.BuildTime),
		slog.String("component", "server"),
	)
	serverLogger.Info("MCP server started")

	server := mcpserver.NewMCPServer(cfg, logger, cmdConfig.Version)
	if err := server.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize the server: %w", err)
	}

	return server.Start()
}

func initSlogLogger(cfg *config.ServerConfig) *slog.Logger {
	var handler slog.Handler

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

func addPersistentFlags(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().String("config", "", "path to the configuration file")
	rootCmd.PersistentFlags().String("host", "localhost", "host of the server")
	rootCmd.PersistentFlags().Int("port", 8080, "port of the server")
	rootCmd.PersistentFlags().String("log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().String("log-format", "json", "log format (json, text)")
	rootCmd.PersistentFlags().String("dokku-path", "/usr/bin/dokku", "path to the Dokku executable")
	rootCmd.PersistentFlags().Bool("cache-enabled", true, "enable the cache of the responses")
	rootCmd.PersistentFlags().Duration("cache-ttl", 5*time.Minute, "cache TTL")
}

// bindFlags binds the command line flags to the configuration viper
// Returns an error if the binding fails for any of the flags
func bindFlags(rootCmd *cobra.Command) error {
	flagBindings := []struct {
		key  string
		flag string
	}{
		{"host", "host"},
		{"port", "port"},
		{"log_level", "log-level"},
		{"log_format", "log-format"},
		{"dokku_path", "dokku-path"},
		{"cache_enabled", "cache-enabled"},
		{"cache_ttl", "cache-ttl"},
	}

	for _, binding := range flagBindings {
		if err := viper.BindPFlag(binding.key, rootCmd.PersistentFlags().Lookup(binding.flag)); err != nil {
			return fmt.Errorf("échec de la liaison du flag '%s': %w", binding.flag, err)
		}
	}

	return nil
}

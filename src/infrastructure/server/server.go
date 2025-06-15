package server

import (
	"fmt"
	"log/slog"

	"github.com/alex-galey/dokku-mcp/src/application/handlers"
	"github.com/alex-galey/dokku-mcp/src/application/prompts"
	"github.com/alex-galey/dokku-mcp/src/infrastructure/config"
	"github.com/alex-galey/dokku-mcp/src/infrastructure/dokku"
	"github.com/mark3labs/mcp-go/server"
)

type MCPServer struct {
	config    *config.ServerConfig
	logger    *slog.Logger
	mcpServer *server.MCPServer
}

func NewMCPServer(config *config.ServerConfig, logger *slog.Logger, version string) *MCPServer {
	mcpServer := server.NewMCPServer(
		"Dokku MCP Server",
		version,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(false, true),
		server.WithPromptCapabilities(true),
	)

	return &MCPServer{
		config:    config,
		logger:    logger,
		mcpServer: mcpServer,
	}
}

func (s *MCPServer) Initialize() error {
	s.logger.Info("Initializing the server MCP components")

	dokkuClient, err := s.initializeDokkuClient()
	if err != nil {
		return fmt.Errorf("failed to initialize the Dokku client: %w", err)
	}

	appRepository := dokku.NewApplicationRepository(dokkuClient, s.logger)
	appHandler := handlers.NewApplicationHandler(appRepository, s.logger)

	if err := s.registerApplicationComponents(appHandler); err != nil {
		return fmt.Errorf("failed to register the application components: %w", err)
	}

	if err := s.registerPrompts(); err != nil {
		return fmt.Errorf("failed to register the prompts: %w", err)
	}

	s.logger.Info("Server MCP components initialized successfully")
	return nil
}

func (s *MCPServer) Start() error {
	s.logger.Info("Server MCP started and listening for connections")

	if err := server.ServeStdio(s.mcpServer); err != nil {
		return fmt.Errorf("failed to start the server MCP: %w", err)
	}

	return nil
}

func (s *MCPServer) initializeDokkuClient() (dokku.DokkuClient, error) {
	dokkuConfig := &dokku.ClientConfig{
		DokkuHost:       s.config.DokkuHost,
		DokkuPort:       s.config.DokkuPort,
		DokkuUser:       s.config.DokkuUser,
		DokkuPath:       s.config.DokkuPath,
		SSHKeyPath:      s.config.SSHKeyPath,
		CommandTimeout:  s.config.Timeout,
		AllowedCommands: s.config.AllowedCommands,
	}

	dokkuClient := dokku.NewDokkuClient(dokkuConfig, s.logger)
	return dokkuClient, nil
}

func (s *MCPServer) registerApplicationComponents(appHandler *handlers.ApplicationHandler) error {
	if err := appHandler.RegisterResources(s.mcpServer); err != nil {
		return fmt.Errorf("failed to register the resources: %w", err)
	}

	if err := appHandler.RegisterTools(s.mcpServer); err != nil {
		return fmt.Errorf("failed to register the tools: %w", err)
	}

	return nil
}

func (s *MCPServer) registerPrompts() error {
	promptsManager := prompts.NewPromptsManager(s.logger)
	return promptsManager.RegisterPrompts(s.mcpServer)
}

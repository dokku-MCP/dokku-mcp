package server

import (
	"fmt"
	"log/slog"

	"github.com/alex-galey/dokku-mcp/internal/domain"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// CorePlugin implements the FeaturePlugin interface and provides essential
// Dokku functionality that is always available regardless of plugins.
type CorePlugin struct {
	mcpHandler domain.MCPHandler
	logger     *slog.Logger

	registeredResources []string
	registeredTools     []string
}

// NewCorePlugin creates a new CorePlugin instance.
func NewCorePlugin(
	mcpHandler domain.MCPHandler,
	logger *slog.Logger,
) domain.FeaturePlugin {
	return &CorePlugin{
		mcpHandler:          mcpHandler,
		logger:              logger,
		registeredResources: make([]string, 0),
		registeredTools:     make([]string, 0),
	}
}

// Name returns the unique name of the plugin.
func (m *CorePlugin) Name() string {
	return "core"
}

// DokkuPluginName returns an empty string since core functionality
// is always available and doesn't depend on any specific plugin.
func (m *CorePlugin) DokkuPluginName() string {
	return ""
}

// Register registers the plugin's resources and tools with the MCP server.
func (m *CorePlugin) Register(mcpServer *server.MCPServer) error {
	m.logger.Debug("Registering core plugin resources and tools")

	if err := m.registerResources(mcpServer); err != nil {
		return fmt.Errorf("failed to register core resources: %w", err)
	}

	if err := m.registerTools(mcpServer); err != nil {
		return fmt.Errorf("failed to register core tools: %w", err)
	}

	m.logger.Debug("Core plugin registered successfully")
	return nil
}

// Deregister removes the plugin's resources and tools from the MCP server.
func (m *CorePlugin) Deregister(mcpServer *server.MCPServer) error {
	m.logger.Debug("Deregistering core plugin resources and tools")

	// Note: The current mcp-go library doesn't provide deregistration methods.
	// This is a placeholder for when the library supports dynamic deregistration.
	// For now, we'll track what was registered for future implementation.

	m.registeredResources = make([]string, 0)
	m.registeredTools = make([]string, 0)

	m.logger.Debug("Core plugin deregistered successfully")
	return nil
}

// registerResources registers the core MCP resources.
func (m *CorePlugin) registerResources(mcpServer *server.MCPServer) error {
	m.logger.Debug("Registering core application resources")

	applicationsResource := mcp.NewResource(
		"dokku://apps",
		"Applications Dokku",
		mcp.WithResourceDescription("Dokku apps resources"),
		mcp.WithMIMEType("application/json"),
	)

	mcpServer.AddResource(applicationsResource, m.mcpHandler.HandleApplicationsResource)
	m.registeredResources = append(m.registeredResources, "dokku://apps")

	m.logger.Debug("Core application resources registered successfully")
	return nil
}

// registerTools registers the core MCP tools.
func (m *CorePlugin) registerTools(mcpServer *server.MCPServer) error {
	m.logger.Debug("Registering core application tools")

	// Tool for creating applications
	createTool := mcp.NewTool(
		"create_application",
		mcp.WithDescription("Create a new Dokku application"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the application to create"),
			mcp.Pattern("^[a-z0-9][a-z0-9-]*[a-z0-9]$"),
		),
	)
	mcpServer.AddTool(createTool, m.mcpHandler.HandleCreateApplication)
	m.registeredTools = append(m.registeredTools, "create_application")

	// Tool for deploying applications
	deployTool := mcp.NewTool(
		"deploy_application",
		mcp.WithDescription("Deploy a Dokku application"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the application to deploy"),
		),
		mcp.WithString("git_ref",
			mcp.Description("Git reference to deploy (optional)"),
			mcp.DefaultString("main"),
		),
	)
	mcpServer.AddTool(deployTool, m.mcpHandler.HandleDeployApplication)
	m.registeredTools = append(m.registeredTools, "deploy_application")

	// Tool for scaling applications
	scaleTool := mcp.NewTool(
		"scale_application",
		mcp.WithDescription("Scale a Dokku application"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the application to scale"),
		),
		mcp.WithString("process_type",
			mcp.Description("Process type (web, worker, etc.)"),
			mcp.DefaultString("web"),
		),
		mcp.WithNumber("scale",
			mcp.Required(),
			mcp.Description("Number of instances"),
			mcp.Min(0),
		),
	)
	mcpServer.AddTool(scaleTool, m.mcpHandler.HandleScaleApplication)
	m.registeredTools = append(m.registeredTools, "scale_application")

	// Tool for configuring applications
	configTool := mcp.NewTool(
		"set_application_config",
		mcp.WithDescription("Set the environment variables of a Dokku application"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the application to set the config for"),
		),
		mcp.WithObject("config",
			mcp.Required(),
			mcp.Description("Environment variables to set"),
			mcp.AdditionalProperties(map[string]interface{}{
				"type": "string",
			}),
		),
	)
	mcpServer.AddTool(configTool, m.mcpHandler.HandleSetApplicationConfig)
	m.registeredTools = append(m.registeredTools, "set_application_config")

	m.logger.Debug("Core application tools registered successfully")
	return nil
}

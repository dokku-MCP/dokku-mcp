package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/alex-galey/dokku-mcp/src/domain/application"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ApplicationHandler handles MCP requests for Dokku applications
type ApplicationHandler struct {
	repository application.Repository
	logger     *slog.Logger
}

func NewApplicationHandler(repository application.Repository, logger *slog.Logger) *ApplicationHandler {
	return &ApplicationHandler{
		repository: repository,
		logger:     logger,
	}
}

func (h *ApplicationHandler) RegisterResources(mcpServer *server.MCPServer) error {
	h.logger.Debug("Registering application resources")

	applicationsResource := mcp.NewResource(
		"dokku://applications",
		"Dokku Applications",
		mcp.WithResourceDescription("List of all Dokku applications"),
		mcp.WithMIMEType("application/json"),
	)

	mcpServer.AddResource(applicationsResource, h.handleApplicationsResource)

	h.logger.Debug("Application resources registered successfully")
	return nil
}

func (h *ApplicationHandler) RegisterTools(mcpServer *server.MCPServer) error {
	h.logger.Debug("Registering application tools")

	createTool := mcp.NewTool(
		"create_application",
		mcp.WithDescription("Create a new Dokku application"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the application to create"),
			mcp.Pattern("^[a-z0-9][a-z0-9-]*[a-z0-9]$"),
		),
	)

	mcpServer.AddTool(createTool, h.handleCreateApplication)

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

	mcpServer.AddTool(deployTool, h.handleDeployApplication)

	scaleTool := mcp.NewTool(
		"scale_application",
		mcp.WithDescription("Scale a Dokku application"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Application name"),
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

	mcpServer.AddTool(scaleTool, h.handleScaleApplication)

	configTool := mcp.NewTool(
		"set_application_config",
		mcp.WithDescription("Set application environment variables"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Application name"),
		),
		mcp.WithObject("config",
			mcp.Required(),
			mcp.Description("Environment variables to set"),
			mcp.AdditionalProperties(map[string]interface{}{
				"type": "string",
			}),
		),
	)

	mcpServer.AddTool(configTool, h.handleSetApplicationConfig)

	h.logger.Debug("Application tools registered successfully")
	return nil
}

// Manage the applications resource
func (h *ApplicationHandler) handleApplicationsResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	h.logger.Debug("Processing applications resource request")

	apps, err := h.repository.GetAll(ctx)
	if err != nil {
		h.logger.Error("Failed to retrieve applications",
			"error", err)
		return nil, fmt.Errorf("failed to get applications: %w", err)
	}

	// Convert to JSON
	appsData := make([]map[string]interface{}, len(apps))
	for i, app := range apps {
		appsData[i] = h.applicationToMap(app)
	}

	jsonData, err := json.MarshalIndent(appsData, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "dokku://applications",
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// Handles the create application tool
func (h *ApplicationHandler) handleCreateApplication(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	h.logger.Debug("Processing create application request")

	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("The 'name' parameter is required and must be a string"), nil
	}

	app, err := application.NewApplication(name)
	if err != nil {
		h.logger.Error("Failed to create application entity",
			"error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create application: %v", err)), nil
	}

	if err := h.repository.Save(ctx, app); err != nil {
		h.logger.Error("Failed to save application",
			"error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to save application: %v", err)), nil
	}

	h.logger.Info("Application created successfully",
		"app_name", name)
	return mcp.NewToolResultText(fmt.Sprintf("✅ Application '%s' created successfully", name)), nil
}

// Handles the deploy application tool
func (h *ApplicationHandler) handleDeployApplication(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	h.logger.Debug("Processing deploy application request")

	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("The 'name' parameter is required"), nil
	}

	gitRef := req.GetString("git_ref", "main")

	app, err := h.repository.GetByName(ctx, name)
	if err != nil {
		h.logger.Error("Échec de récupération de l'application",
			"erreur", err)
		return mcp.NewToolResultError(fmt.Sprintf("Application not found: %v", err)), nil
	}

	// Simulate the deployment (in a real implementation, this would interact with Dokku)
	if err := app.Deploy(gitRef, "", ""); err != nil {
		h.logger.Error("Failed to deploy",
			"error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Deployment failed: %v", err)), nil
	}

	if err := h.repository.Save(ctx, app); err != nil {
		h.logger.Warn("Échec de sauvegarde de l'état après déploiement",
			"erreur", err)
	}

	h.logger.Info("Application déployée avec succès",
		"nom_app", name,
		"git_ref", gitRef)

	return mcp.NewToolResultText(fmt.Sprintf("✅ Application '%s' deployed successfully (ref: %s)", name, gitRef)), nil
}

// handleScaleApplication gère l'outil de mise à l'échelle d'application
func (h *ApplicationHandler) handleScaleApplication(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	h.logger.Debug("Processing scale application request")

	// Extraire les paramètres avec la bonne API
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("The 'name' parameter is required"), nil
	}

	processType := req.GetString("process_type", "web")

	scale, err := req.RequireFloat("scale")
	if err != nil {
		return mcp.NewToolResultError("The 'scale' parameter is required and must be a number"), nil
	}
	scaleInt := int(scale)

	// Récupérer l'application
	app, err := h.repository.GetByName(ctx, name)
	if err != nil {
		h.logger.Error("Échec de récupération de l'application",
			"erreur", err)
		return mcp.NewToolResultError(fmt.Sprintf("Application not found: %v", err)), nil
	}

	// Mapper le type de processus
	var mappedType application.ProcessType
	switch processType {
	case "web":
		mappedType = application.ProcessTypeWeb
	case "worker":
		mappedType = application.ProcessTypeWorker
	case "cron":
		mappedType = application.ProcessTypeCron
	case "release":
		mappedType = application.ProcessTypeRelease
	default:
		mappedType = application.ProcessTypeWeb
	}

	// Créer ou mettre à jour le processus
	process, err := application.NewProcess(mappedType, "", scaleInt)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create process: %v", err)), nil
	}

	if err := app.Config().AddProcess(process); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add process: %v", err)), nil
	}

	if err := h.repository.Save(ctx, app); err != nil {
		h.logger.Error("Échec de sauvegarde après mise à l'échelle",
			"erreur", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to save: %v", err)), nil
	}

	h.logger.Info("Application mise à l'échelle avec succès",
		"nom_app", name,
		"type_process", processType,
		"échelle", scaleInt)

	return mcp.NewToolResultText(fmt.Sprintf("✅ Application '%s' scaled: %s=%d", name, processType, scaleInt)), nil
}

func (h *ApplicationHandler) handleSetApplicationConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	h.logger.Debug("Processing set application config request")

	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("The 'name' parameter is required"), nil
	}

	args := req.GetArguments()
	configData, ok := args["config"].(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("The 'config' parameter is required and must be an object"), nil
	}

	app, err := h.repository.GetByName(ctx, name)
	if err != nil {
		h.logger.Error("Échec de récupération de l'application",
			"erreur", err)
		return mcp.NewToolResultError(fmt.Sprintf("Application not found: %v", err)), nil
	}

	// Update the configuration
	appConfig := app.Config()
	for key, value := range configData {
		if valueStr, ok := value.(string); ok {
			if err := appConfig.SetEnvironmentVar(key, valueStr); err != nil {
				h.logger.Warn("Failed to set variable",
					"error", err,
					"key", key)
				return mcp.NewToolResultError(fmt.Sprintf("Failed to set variable %s: %v", key, err)), nil
			}
		}
	}

	if err := h.repository.Save(ctx, app); err != nil {
		h.logger.Error("Failed to save after configuration update",
			"error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to save: %v", err)), nil
	}

	h.logger.Info("Application configuration updated successfully",
		"app_name", name,
		"number_config", len(configData))

	return mcp.NewToolResultText(fmt.Sprintf("✅ Application '%s' configuration updated (%d variables)", name, len(configData))), nil
}

// applicationToMap converts an application to a map for JSON serialization
func (h *ApplicationHandler) applicationToMap(app *application.Application) map[string]interface{} {
	config := app.Config()

	// Convert processes to map
	processes := make(map[string]interface{})
	for processType, process := range config.Processes {
		processes[string(processType)] = map[string]interface{}{
			"command": process.Command,
			"scale":   process.Scale,
		}
	}

	return map[string]interface{}{
		"name":        app.Name(),
		"state":       string(app.State()),
		"created_at":  app.CreatedAt(),
		"updated_at":  app.UpdatedAt(),
		"last_deploy": app.LastDeploy(),
		"git_ref":     app.GitRef(),
		"build_image": app.BuildImage(),
		"run_image":   app.RunImage(),
		"config": map[string]interface{}{
			"buildpack":       config.BuildPack,
			"domains":         config.Domains,
			"environment":     config.EnvironmentVars,
			"processes":       processes,
			"resource_limits": config.ResourceLimits,
		},
	}
}

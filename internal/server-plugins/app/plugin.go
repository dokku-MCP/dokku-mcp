package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	dokkuApi "github.com/alex-galey/dokku-mcp/internal/dokku-api"
	"github.com/alex-galey/dokku-mcp/internal/server-plugin/domain"
	appusecases "github.com/alex-galey/dokku-mcp/internal/server-plugins/app/application"
	appdomain "github.com/alex-galey/dokku-mcp/internal/server-plugins/app/domain"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/app/infrastructure"
	"github.com/alex-galey/dokku-mcp/internal/shared"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/fx"
)

// AppsServerPlugin implements the unified ServerPlugin interface for Dokku applications
// This replaces the legacy AppsPlugin and demonstrates the new architecture
type AppsServerPlugin struct {
	applicationUseCase *appusecases.ApplicationUseCase
	logger             *slog.Logger
}

// NewAppsServerPlugin creates a new unified apps server plugin
func NewAppsServerPlugin(
	applicationRepo appdomain.ApplicationRepository,
	deploymentSvc shared.DeploymentService,
	logger *slog.Logger,
) domain.ServerPlugin {
	return &AppsServerPlugin{
		applicationUseCase: appusecases.NewApplicationUseCase(applicationRepo, deploymentSvc, logger),
		logger:             logger,
	}
}

// ServerPlugin interface implementation
func (p *AppsServerPlugin) ID() string   { return "apps" }
func (p *AppsServerPlugin) Name() string { return "Dokku Applications" }

func (p *AppsServerPlugin) Description() string {
	return "Comprehensive Dokku application management including deployment, scaling, and configuration"
}

func (p *AppsServerPlugin) Version() string { return "0.2.0" }

// Core apps functionality - no specific plugin dependency
func (p *AppsServerPlugin) DokkuPluginName() string { return "" }

// ResourceProvider implementation
func (p *AppsServerPlugin) GetResources(ctx context.Context) ([]domain.Resource, error) {
	return []domain.Resource{
		{
			URI:         "dokku://apps/list",
			Name:        "Application List",
			Description: "Complete list of all Dokku applications with status",
			MIMEType:    "application/json",
			Handler:     p.handleApplicationListResource,
		},
	}, nil
}

// ToolProvider implementation
func (p *AppsServerPlugin) GetTools(ctx context.Context) ([]domain.Tool, error) {
	return []domain.Tool{
		{
			Name:        "create_app",
			Description: "Create a new Dokku application with validation",
			Builder:     p.buildCreateAppTool,
			Handler:     p.handleCreateApp,
		},
		{
			Name:        "deploy_app",
			Description: "Deploy application from Git with options",
			Builder:     p.buildDeployAppTool,
			Handler:     p.handleDeployApp,
		},
		{
			Name:        "scale_app",
			Description: "Scale application processes with validation",
			Builder:     p.buildScaleAppTool,
			Handler:     p.handleScaleApp,
		},
		{
			Name:        "configure_app",
			Description: "Set environment variables with validation",
			Builder:     p.buildConfigureAppTool,
			Handler:     p.handleConfigureApp,
		},
		{
			Name:        "get_app_status",
			Description: "Get comprehensive application status",
			Builder:     p.buildGetAppStatusTool,
			Handler:     p.handleGetAppStatus,
		},
	}, nil
}

// PromptProvider implementation
func (p *AppsServerPlugin) GetPrompts(ctx context.Context) ([]domain.Prompt, error) {
	return []domain.Prompt{
		{
			Name:        "app_doctor",
			Description: "Comprehensive application health diagnosis and troubleshooting",
			Builder:     p.buildAppDoctorPrompt,
			Handler:     p.handleAppDoctorPrompt,
		},
	}, nil
}

// Resource handlers
func (p *AppsServerPlugin) handleApplicationListResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	applications, err := p.applicationUseCase.GetAllApplications(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve applications: %w", err)
	}

	apps := make([]appdomain.ApplicationInfo, len(applications))
	for i, app := range applications {
		apps[i] = appdomain.ApplicationInfo{
			Name:       app.Name().Value(),
			State:      string(app.State().Value()),
			IsRunning:  app.IsRunning(),
			IsDeployed: app.IsDeployed(),
			CreatedAt:  app.CreatedAt(),
			UpdatedAt:  app.UpdatedAt(),
		}
	}

	data := appdomain.ApplicationListData{
		Applications: apps,
		Count:        len(apps),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize applications: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// Tool builders
func (p *AppsServerPlugin) buildCreateAppTool() mcp.Tool {
	return mcp.NewTool(
		"create_app",
		mcp.WithDescription("Create a new Dokku application with comprehensive validation"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Application name (lowercase, alphanumeric, hyphens allowed)"),
			mcp.Pattern("^[a-z0-9-]+$"),
		),
		mcp.WithString("buildpack",
			mcp.Description("Specific buildpack to use (optional)"),
		),
		mcp.WithBoolean("no_vhost",
			mcp.Description("Disable default vhost creation"),
		),
	)
}

func (p *AppsServerPlugin) buildDeployAppTool() mcp.Tool {
	return mcp.NewTool(
		"deploy_app",
		mcp.WithDescription("Deploy application from Git repository"),
		mcp.WithString("app_name",
			mcp.Required(),
			mcp.Description("Name of the application to deploy"),
		),
		mcp.WithString("repo_url",
			mcp.Required(),
			mcp.Description("URL of the Git repository to deploy from"),
		),
		mcp.WithString("git_ref",
			mcp.Description("Git reference to deploy (branch, tag, or commit)"),
		),
		mcp.WithBoolean("force",
			mcp.Description("Force deployment even if no changes detected"),
		),
	)
}

func (p *AppsServerPlugin) buildScaleAppTool() mcp.Tool {
	return mcp.NewTool(
		"scale_app",
		mcp.WithDescription("Scale application processes"),
		mcp.WithString("app_name",
			mcp.Required(),
			mcp.Description("Name of the application to scale"),
		),
		mcp.WithString("process_type",
			mcp.Description("Process type to scale (web, worker, etc.)"),
		),
		mcp.WithNumber("instances",
			mcp.Required(),
			mcp.Description("Number of instances to scale to"),
		),
	)
}

func (p *AppsServerPlugin) buildConfigureAppTool() mcp.Tool {
	return mcp.NewTool(
		"configure_app",
		mcp.WithDescription("Set environment variables for an application"),
		mcp.WithString("app_name",
			mcp.Required(),
			mcp.Description("Name of the application to configure"),
		),
		mcp.WithObject("config",
			mcp.Required(),
			mcp.Description("Environment variables as key-value pairs"),
			mcp.Properties(map[string]interface{}{ // NOTE: This is a valid exception
				"additionalProperties": map[string]interface{}{ // NOTE: This is a valid exception
					"type": "string",
				},
			}),
		),
	)
}

func (p *AppsServerPlugin) buildGetAppStatusTool() mcp.Tool {
	return mcp.NewTool(
		"get_app_status",
		mcp.WithDescription("Get comprehensive status information for an application"),
		mcp.WithString("app_name",
			mcp.Required(),
			mcp.Description("Name of the application"),
		),
	)
}

// Tool handlers
func (p *AppsServerPlugin) handleCreateApp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("Application name is required"), nil
	}

	cmd := appusecases.CreateApplicationCommand{Name: name}
	if err := p.applicationUseCase.CreateApplication(ctx, cmd); err != nil {
		if errors.Is(err, appdomain.ErrApplicationAlreadyExists) {
			return mcp.NewToolResultError(fmt.Sprintf("Application '%s' already exists", name)), nil
		}
		if errors.Is(err, appdomain.ErrInvalidApplicationName) {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid application name '%s'", name)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create application: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Application '%s' created successfully", name)), nil
}

func (p *AppsServerPlugin) handleDeployApp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appName, err := req.RequireString("app_name")
	if err != nil {
		return mcp.NewToolResultError("Application name is required"), nil
	}

	repoURL, err := req.RequireString("repo_url")
	if err != nil {
		return mcp.NewToolResultError("Repository URL is required"), nil
	}

	gitRef := "main"
	if gitRefParam, ok := req.GetArguments()["git_ref"]; ok {
		if gitRefStr, ok := gitRefParam.(string); ok && gitRefStr != "" {
			gitRef = gitRefStr
		}
	}

	cmd := appusecases.DeployApplicationCommand{
		Name:    appName,
		RepoURL: repoURL,
		GitRef:  gitRef,
	}

	if err := p.applicationUseCase.DeployApplication(ctx, cmd); err != nil {
		if errors.Is(err, appdomain.ErrApplicationNotFound) {
			return mcp.NewToolResultError(fmt.Sprintf("Application '%s' not found", appName)), nil
		}
		if errors.Is(err, appdomain.ErrDeploymentInProgress) {
			return mcp.NewToolResultError(fmt.Sprintf("Deployment already in progress for '%s'", appName)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("Failed to deploy application: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Application '%s' deployed successfully from '%s'", appName, gitRef)), nil
}

func (p *AppsServerPlugin) handleScaleApp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appName, err := req.RequireString("app_name")
	if err != nil {
		return mcp.NewToolResultError("Application name is required"), nil
	}

	processType := "web"
	if processTypeParam, ok := req.GetArguments()["process_type"]; ok {
		if processTypeStr, ok := processTypeParam.(string); ok && processTypeStr != "" {
			processType = processTypeStr
		}
	}

	instancesParam, ok := req.GetArguments()["instances"]
	if !ok {
		return mcp.NewToolResultError("Number of instances is required"), nil
	}

	var instances int
	switch v := instancesParam.(type) {
	case float64:
		instances = int(v)
	case int:
		instances = v
	default:
		return mcp.NewToolResultError("Invalid instances value - must be a number"), nil
	}

	cmd := appusecases.ScaleApplicationCommand{
		Name:        appName,
		ProcessType: processType,
		Scale:       instances,
	}

	if err := p.applicationUseCase.ScaleApplication(ctx, cmd); err != nil {
		if errors.Is(err, appdomain.ErrApplicationNotFound) {
			return mcp.NewToolResultError(fmt.Sprintf("Application '%s' not found", appName)), nil
		}
		if errors.Is(err, appdomain.ErrApplicationNotDeployed) {
			return mcp.NewToolResultError(fmt.Sprintf("Application '%s' is not deployed", appName)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("Failed to scale application: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Application '%s' scaled to %d instances for process type '%s'", appName, instances, processType)), nil
}

func (p *AppsServerPlugin) handleConfigureApp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appName, err := req.RequireString("app_name")
	if err != nil {
		return mcp.NewToolResultError("Application name is required"), nil
	}

	configVars := make(map[string]string)
	if configParam, ok := req.GetArguments()["config"]; ok {
		if configMap, ok := configParam.(map[string]interface{}); ok { // NOTE: This is a valid exception
			for key, value := range configMap {
				if valueStr, ok := value.(string); ok {
					configVars[key] = valueStr
				}
			}
		}
	}

	if len(configVars) == 0 {
		return mcp.NewToolResultError("At least one configuration variable is required"), nil
	}

	cmd := appusecases.SetConfigCommand{
		Name:   appName,
		Config: configVars,
	}

	if err := p.applicationUseCase.SetApplicationConfig(ctx, cmd); err != nil {
		if errors.Is(err, appdomain.ErrApplicationNotFound) {
			return mcp.NewToolResultError(fmt.Sprintf("Application '%s' not found", appName)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("Failed to configure application: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Application '%s' configured successfully with %d variables", appName, len(configVars))), nil
}

func (p *AppsServerPlugin) handleGetAppStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appName, err := req.RequireString("app_name")
	if err != nil {
		return mcp.NewToolResultError("Application name is required"), nil
	}

	app, err := p.applicationUseCase.GetApplicationByName(ctx, appName)
	if err != nil {
		if errors.Is(err, appdomain.ErrApplicationNotFound) {
			return mcp.NewToolResultError(fmt.Sprintf("Application '%s' not found", appName)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get application status: %v", err)), nil
	}

	status := appdomain.ApplicationStatus{
		Name:       app.Name().Value(),
		State:      string(app.State().Value()),
		CreatedAt:  app.CreatedAt(),
		UpdatedAt:  app.UpdatedAt(),
		IsRunning:  app.IsRunning(),
		IsDeployed: app.IsDeployed(),
		Domains:    app.GetDomains(),
	}

	statusJSON, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("Failed to serialize status"), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Application Status for '%s':\n%s", appName, string(statusJSON))), nil
}

// Prompt implementations
func (p *AppsServerPlugin) buildAppDoctorPrompt() mcp.Prompt {
	return mcp.NewPrompt(
		"app_doctor",
		mcp.WithPromptDescription("Comprehensive application health diagnosis and troubleshooting"),
		mcp.WithArgument("app_name",
			mcp.RequiredArgument(),
			mcp.ArgumentDescription("Name of the Dokku application to diagnose"),
		),
	)
}

func (p *AppsServerPlugin) handleAppDoctorPrompt(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	// Extract required argument from request params (Arguments is a typed map)
	appName, ok := req.Params.Arguments["app_name"]
	if !ok || appName == "" {
		return &mcp.GetPromptResult{
			Description: "app_name parameter is required",
		}, fmt.Errorf("app_name parameter is required")
	}

	// Use the diagnostic template
	tmpl := NewApplicationPromptTemplates().GetDiagnosticPrompt()
	promptText := fmt.Sprintf(tmpl.Template, appName)

	return &mcp.GetPromptResult{
		Description: tmpl.Description,
		Messages: []mcp.PromptMessage{
			{
				Role:    "user",
				Content: mcp.TextContent{Type: "text", Text: promptText},
			},
		},
	}, nil
}

var Module = fx.Module("app",
	fx.Provide(
		// Provide the infrastructure layer dependencies
		fx.Annotate(
			func(client dokkuApi.DokkuClient, logger *slog.Logger) appdomain.ApplicationRepository {
				return infrastructure.NewDokkuApplicationRepository(client, logger)
			},
		),
		// Provide the main plugin - deployment service will be injected from deployment plugin
		fx.Annotate(
			NewAppsServerPlugin,
			fx.As(new(domain.ServerPlugin)),
			fx.ResultTags(`group:"server_plugins"`),
		),
	),
)

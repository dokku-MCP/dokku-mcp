package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/server"

	dokkuApi "github.com/dokku-mcp/dokku-mcp/internal/dokku-api"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	appusecases "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/app/application"
	appdomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/app/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/app/infrastructure"
	"github.com/dokku-mcp/dokku-mcp/internal/shared"
	"github.com/dokku-mcp/dokku-mcp/pkg/config"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/fx"
)

// AppsServerPlugin implements the unified ServerPlugin interface for Dokku applications
// This replaces the legacy AppsPlugin and demonstrates the new architecture
type AppsServerPlugin struct {
	applicationUseCase *appusecases.ApplicationUseCase
	logger             *slog.Logger
	logsConfig         config.LogsConfig
}

// NewAppsServerPlugin creates a new unified apps server plugin
func NewAppsServerPlugin(
	applicationRepo appdomain.ApplicationRepository,
	deploymentSvc shared.DeploymentService,
	logger *slog.Logger,
	logsConfig config.LogsConfig,
) domain.ServerPlugin {
	return &AppsServerPlugin{
		applicationUseCase: appusecases.NewApplicationUseCase(applicationRepo, deploymentSvc, logger),
		logger:             logger,
		logsConfig:         logsConfig,
	}
}

// ServerPlugin interface implementation
func (p *AppsServerPlugin) ID() string   { return "apps" }
func (p *AppsServerPlugin) Name() string { return "Dokku Applications" }

func (p *AppsServerPlugin) Description() string {
	return "Comprehensive Dokku application management including deployment, scaling, and configuration"
}

func (p *AppsServerPlugin) Version() string { return "0.3.0" }

// Core apps functionality - no specific plugin dependency
func (p *AppsServerPlugin) DokkuPluginName() string { return "" }

// ResourceProvider implementation
func (p *AppsServerPlugin) GetResources(ctx context.Context) ([]domain.Resource, error) {
	// Get application list
	applications, err := p.applicationUseCase.GetAllApplications(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve applications for resources: %w", err)
	}

	resources := []domain.Resource{
		{
			URI:         "dokku://apps/list",
			Name:        "Application List",
			Description: "Complete list of all Dokku applications with status",
			MIMEType:    "application/json",
			Handler:     p.handleApplicationListResource,
		},
	}

	// Add runtime logs resources for each application
	for _, app := range applications {
		resources = append(resources, domain.Resource{
			URI:         fmt.Sprintf("dokku://app/%s/logs", app.Name().Value()),
			Name:        fmt.Sprintf("Runtime Logs: %s", app.Name().Value()),
			Description: fmt.Sprintf("Application runtime logs for %s", app.Name().Value()),
			MIMEType:    "application/json",
			Handler:     p.handleRuntimeLogsResource,
		})
	}

	return resources, nil
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
		{
			Name:        "get_runtime_logs",
			Description: "Retrieve runtime logs from a Dokku application",
			Builder:     p.buildGetRuntimeLogsTool,
			Handler:     p.handleGetRuntimeLogs,
		},
		{
			Name:        "follow_runtime_logs",
			Description: "Follow new runtime log lines for a short period",
			Builder:     buildFollowRuntimeLogsTool,
			Handler:     p.handleFollowRuntimeLogs,
		},
		{
			Name:        "get_failed_deploy_logs",
			Description: "Retrieve logs of the last failed deploy",
			Builder:     buildGetFailedDeployLogsTool,
			Handler:     p.handleGetFailedDeployLogs,
		},
		{
			Name:        "restart_app",
			Description: "Restart all processes of an application",
			Builder:     buildProcessStateTool("restart_app", "Restart application", "Restart all processes of an application", false),
			Handler:     p.processStateHandler(appdomain.ProcessRestart),
		},
		{
			Name:        "stop_app",
			Description: "Stop all processes of an application",
			Builder:     buildProcessStateTool("stop_app", "Stop application", "Stop all processes of an application; it goes offline until start_app", true),
			Handler:     p.processStateHandler(appdomain.ProcessStop),
		},
		{
			Name:        "start_app",
			Description: "Start a stopped application",
			Builder:     buildProcessStateTool("start_app", "Start application", "Start all processes of a stopped application", false),
			Handler:     p.processStateHandler(appdomain.ProcessStart),
		},
		{
			Name:        "rollback_app",
			Description: "Redeploy an earlier Git reference",
			Builder:     buildRollbackAppTool,
			Handler:     p.handleRollbackApp,
		},
	}, nil
}

const (
	defaultFollowSeconds = 15
	maxFollowSeconds     = 60
	maxFollowLines       = 2000
)

// FollowedLogs is the structured result of follow_runtime_logs.
type FollowedLogs struct {
	AppName   string   `json:"app_name"`
	Seconds   int      `json:"seconds" jsonschema:"How long the logs were followed"`
	Lines     []string `json:"lines"`
	Truncated bool     `json:"truncated" jsonschema:"True when max_lines was reached before the time ran out"`
}

func buildFollowRuntimeLogsTool() mcp.Tool {
	return mcp.NewTool(
		"follow_runtime_logs",
		mcp.WithTitleAnnotation("Follow runtime logs"),
		mcp.WithDescription("Watch an application's logs live for a few seconds, e.g. while reproducing a request or after a deploy. "+
			"Lines are also streamed as progress notifications when the client sends a progress token. "+
			"Use get_runtime_logs for past logs"),
		mcp.WithString("app_name", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithInteger("seconds",
			mcp.Description(fmt.Sprintf("How long to follow the logs (default %d, max %d)", defaultFollowSeconds, maxFollowSeconds)),
			mcp.Min(1), mcp.Max(maxFollowSeconds),
		),
		mcp.WithInteger("max_lines",
			mcp.Description(fmt.Sprintf("Stop after this many lines (default and max %d)", maxFollowLines)),
			mcp.Min(1), mcp.Max(maxFollowLines),
		),
		mcp.WithOutputSchema[FollowedLogs](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func (p *AppsServerPlugin) handleFollowRuntimeLogs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appName, err := req.RequireString("app_name")
	if err != nil {
		return mcp.NewToolResultError("Application name is required"), nil
	}
	seconds := max(1, min(req.GetInt("seconds", defaultFollowSeconds), maxFollowSeconds))
	maxLines := max(1, min(req.GetInt("max_lines", maxFollowLines), maxFollowLines))

	result, err := p.applicationUseCase.FollowLogs(ctx, appName, time.Duration(seconds)*time.Second, maxLines, progressReporter(ctx, req))
	if err != nil {
		return appErrorResult(appName, "follow logs of", err), nil
	}
	return mcp.NewToolResultStructuredOnly(FollowedLogs{
		AppName:   appName,
		Seconds:   seconds,
		Lines:     result.Lines,
		Truncated: result.Truncated,
	}), nil
}

// progressReporter returns a callback that forwards each line to the client
// as a progress notification, or nil when the client did not ask for
// progress.
func progressReporter(ctx context.Context, req mcp.CallToolRequest) func(string) {
	if req.Params.Meta == nil || req.Params.Meta.ProgressToken == nil {
		return nil
	}
	srv := server.ServerFromContext(ctx)
	if srv == nil {
		return nil
	}
	token := req.Params.Meta.ProgressToken
	count := 0
	return func(line string) {
		count++
		_ = srv.SendNotificationToClient(ctx, string(mcp.MethodNotificationProgress), map[string]any{
			"progressToken": token,
			"progress":      count,
			"message":       line,
		})
	}
}

func buildGetFailedDeployLogsTool() mcp.Tool {
	return mcp.NewTool(
		"get_failed_deploy_logs",
		mcp.WithTitleAnnotation("Get failed deploy logs"),
		mcp.WithDescription("Retrieve the logs of the containers from an application's last failed deploy, "+
			"e.g. when a new release crashed at boot or failed its healthchecks"),
		mcp.WithString("app_name", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func buildProcessStateTool(name, title, description string, destructive bool) func() mcp.Tool {
	return func() mcp.Tool {
		return mcp.NewTool(
			name,
			mcp.WithTitleAnnotation(title),
			mcp.WithDescription(description),
			mcp.WithString("app_name", mcp.Required(), mcp.Description("Name of the application")),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(destructive),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(false),
		)
	}
}

func buildRollbackAppTool() mcp.Tool {
	return mcp.NewTool(
		"rollback_app",
		mcp.WithTitleAnnotation("Roll back application"),
		mcp.WithDescription("Redeploy an earlier, known-good Git reference. Dokku keeps no release history, so this "+
			"deploys git_ref from the repository of the last deploy_app (or repo_url). Returns a deployment_id "+
			"to follow with get_deployment_status"),
		mcp.WithString("app_name", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithString("git_ref", mcp.Required(), mcp.Description("Branch, tag or commit to redeploy")),
		mcp.WithString("repo_url", mcp.Description("Repository to deploy from; defaults to the last deployed repository")),
		mcp.WithOutputSchema[DeployStarted](),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

func (p *AppsServerPlugin) handleGetFailedDeployLogs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appName, err := req.RequireString("app_name")
	if err != nil {
		return mcp.NewToolResultError("Application name is required"), nil
	}
	logs, err := p.applicationUseCase.GetFailedDeployLogs(ctx, appName)
	if err != nil {
		return appErrorResult(appName, "get failed deploy logs", err), nil
	}
	if strings.TrimSpace(logs) == "" {
		return mcp.NewToolResultText(fmt.Sprintf("No failed deploy containers found for '%s'.", appName)), nil
	}
	return mcp.NewToolResultText(logs), nil
}

func (p *AppsServerPlugin) processStateHandler(action appdomain.ProcessAction) domain.ToolHandler {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		appName, err := req.RequireString("app_name")
		if err != nil {
			return mcp.NewToolResultError("Application name is required"), nil
		}
		if err := p.applicationUseCase.ChangeProcessState(ctx, appName, action); err != nil {
			return appErrorResult(appName, string(action), err), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Application '%s': %s done", appName, action)), nil
	}
}

func (p *AppsServerPlugin) handleRollbackApp(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appName, err := req.RequireString("app_name")
	if err != nil {
		return mcp.NewToolResultError("Application name is required"), nil
	}
	gitRef, err := req.RequireString("git_ref")
	if err != nil {
		return mcp.NewToolResultError("git_ref is required"), nil
	}
	result, err := p.applicationUseCase.RollbackApplication(ctx, appusecases.RollbackApplicationCommand{
		Name:    appName,
		GitRef:  gitRef,
		RepoURL: req.GetString("repo_url", ""),
	})
	if err != nil {
		return appErrorResult(appName, "roll back", err), nil
	}
	return mcp.NewToolResultStructuredOnly(DeployStarted{
		DeploymentID: result.ID,
		AppName:      appName,
		GitRef:       gitRef,
		Status:       string(result.Status),
		Message:      "Rollback started; poll get_deployment_status with this deployment_id until done is true.",
	}), nil
}

// appErrorResult turns a use-case error into a tool error the model can act on.
func appErrorResult(appName, action string, err error) *mcp.CallToolResult {
	switch {
	case errors.Is(err, appdomain.ErrApplicationNotFound):
		return mcp.NewToolResultError(fmt.Sprintf("Application '%s' not found", appName))
	case errors.Is(err, appdomain.ErrDeploymentInProgress):
		return mcp.NewToolResultError(fmt.Sprintf("Deployment already in progress for '%s'", appName))
	default:
		return mcp.NewToolResultError(fmt.Sprintf("Failed to %s '%s': %v", action, appName, err))
	}
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
		mcp.WithTitleAnnotation("Create application"),
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
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func (p *AppsServerPlugin) buildDeployAppTool() mcp.Tool {
	return mcp.NewTool(
		"deploy_app",
		mcp.WithTitleAnnotation("Deploy application"),
		mcp.WithDescription("Deploy an application from a Git repository. Returns immediately with a deployment_id; "+
			"the build continues in the background, so poll get_deployment_status until it is done"),
		mcp.WithString("app_name",
			mcp.Required(),
			mcp.Description("Name of the application to deploy"),
		),
		mcp.WithString("repo_url",
			mcp.Required(),
			mcp.Description("URL of the Git repository to deploy from"),
		),
		mcp.WithString("git_ref",
			mcp.Description("Git reference to deploy (branch, tag, or commit). Defaults to main"),
		),
		mcp.WithOutputSchema[DeployStarted](),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// DeployStarted is the structured result of deploy_app.
type DeployStarted struct {
	DeploymentID string `json:"deployment_id" jsonschema:"Pass to get_deployment_status to follow the build"`
	AppName      string `json:"app_name"`
	GitRef       string `json:"git_ref"`
	Status       string `json:"status"`
	Message      string `json:"message"`
}

func (p *AppsServerPlugin) buildScaleAppTool() mcp.Tool {
	return mcp.NewTool(
		"scale_app",
		mcp.WithTitleAnnotation("Scale application"),
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
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func (p *AppsServerPlugin) buildConfigureAppTool() mcp.Tool {
	return mcp.NewTool(
		"configure_app",
		mcp.WithTitleAnnotation("Set environment variables"),
		mcp.WithDescription("Set environment variables for an application"),
		mcp.WithString("app_name",
			mcp.Required(),
			mcp.Description("Name of the application to configure"),
		),
		mcp.WithObject("config",
			mcp.Required(),
			mcp.Description("Environment variables as key-value pairs. Keys must be valid identifiers; values may contain any characters"),
			mcp.AdditionalProperties(map[string]any{"type": "string"}),
		),
		mcp.WithBoolean("restart",
			mcp.Description("Restart the application so it picks up the new values (default true)"),
			mcp.DefaultBool(true),
		),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func (p *AppsServerPlugin) buildGetAppStatusTool() mcp.Tool {
	return mcp.NewTool(
		"get_app_status",
		mcp.WithTitleAnnotation("Get application status"),
		mcp.WithDescription("Get comprehensive status information for an application"),
		mcp.WithString("app_name",
			mcp.Required(),
			mcp.Description("Name of the application"),
		),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
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

	result, err := p.applicationUseCase.DeployApplication(ctx, cmd)
	if err != nil {
		if errors.Is(err, appdomain.ErrApplicationNotFound) {
			return mcp.NewToolResultError(fmt.Sprintf("Application '%s' not found", appName)), nil
		}
		if errors.Is(err, appdomain.ErrDeploymentInProgress) {
			return mcp.NewToolResultError(fmt.Sprintf("Deployment already in progress for '%s'", appName)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("Failed to deploy application: %v", err)), nil
	}

	return mcp.NewToolResultStructuredOnly(DeployStarted{
		DeploymentID: result.ID,
		AppName:      appName,
		GitRef:       gitRef,
		Status:       string(result.Status),
		Message:      "Code synced; the build is running in the background. Poll get_deployment_status with this deployment_id until done is true.",
	}), nil
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
		if configMap, ok := configParam.(map[string]any); ok { // NOTE: This is a valid exception
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
		Name:    appName,
		Config:  configVars,
		Restart: req.GetBool("restart", true),
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

// Runtime logs resource handler
func (p *AppsServerPlugin) handleRuntimeLogsResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Parse URI to get app name
	uri := req.Params.URI
	if !strings.HasPrefix(uri, "dokku://app/") {
		return nil, fmt.Errorf("invalid runtime logs resource URI: %s", uri)
	}

	// Extract app name and verify it's a logs request
	parts := strings.Split(strings.TrimPrefix(uri, "dokku://app/"), "/")
	if len(parts) < 2 || parts[1] != "logs" {
		return nil, fmt.Errorf("invalid runtime logs resource URI format: %s", uri)
	}

	appName := parts[0]

	logs, err := p.runtimeLogs(ctx, appName, p.logsConfig.Runtime.DefaultLines)
	if err != nil {
		return nil, err
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "text/plain",
			Text:     logs.Logs,
		},
	}, nil
}

// Runtime logs tool builder
func (p *AppsServerPlugin) buildGetRuntimeLogsTool() mcp.Tool {
	return mcp.NewTool(
		"get_runtime_logs",
		mcp.WithTitleAnnotation("Get runtime logs"),
		mcp.WithDescription("Retrieve runtime logs from a Dokku application"),
		mcp.WithString("app_name",
			mcp.Required(),
			mcp.Description("Name of the application"),
		),
		mcp.WithInteger("lines",
			mcp.Description(fmt.Sprintf("Number of log lines to retrieve (default: %d, max: %d)", p.logsConfig.Runtime.DefaultLines, p.logsConfig.Runtime.MaxLines)),
			mcp.Min(1),
		),
		mcp.WithOutputSchema[RuntimeLogs](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

// Runtime logs tool handler
func (p *AppsServerPlugin) handleGetRuntimeLogs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appName, err := req.RequireString("app_name")
	if err != nil {
		return mcp.NewToolResultError("Application name is required"), nil
	}

	logs, err := p.runtimeLogs(ctx, appName, req.GetInt("lines", p.logsConfig.Runtime.DefaultLines))
	if err != nil {
		if errors.Is(err, appdomain.ErrApplicationNotFound) {
			return mcp.NewToolResultError(fmt.Sprintf("Application '%s' not found", appName)), nil
		}
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get logs: %v", err)), nil
	}

	return mcp.NewToolResultStructuredOnly(logs), nil
}

// RuntimeLogs is the structured result of get_runtime_logs.
type RuntimeLogs struct {
	AppName string `json:"app_name"`
	Lines   int    `json:"lines" jsonschema:"Number of lines requested"`
	Logs    string `json:"logs"`
}

// runtimeLogs fetches the last lines of an application's logs, clamping the
// line count to the configured maximum.
func (p *AppsServerPlugin) runtimeLogs(ctx context.Context, appName string, lines int) (RuntimeLogs, error) {
	lines = max(1, min(lines, p.logsConfig.Runtime.MaxLines))
	logs, err := p.applicationUseCase.GetApplicationLogs(ctx, appName, lines)
	if err != nil {
		return RuntimeLogs{}, err
	}
	return RuntimeLogs{AppName: appName, Lines: lines, Logs: logs}, nil
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
			func(
				applicationRepo appdomain.ApplicationRepository,
				deploymentSvc shared.DeploymentService,
				logger *slog.Logger,
				config *config.ServerConfig,
			) domain.ServerPlugin {
				return NewAppsServerPlugin(
					applicationRepo,
					deploymentSvc,
					logger,
					config.Logs,
				)
			},
			fx.As(new(domain.ServerPlugin)),
			fx.ResultTags(`group:"server_plugins"`),
		),
	),
)

package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"time"

	appdomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/app/domain"

	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	deployment_domain "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/deployment/domain"
	"github.com/mark3labs/mcp-go/mcp"
)

// DeploymentServerPlugin implements the ServerPlugin interface for deployment functionality
type DeploymentServerPlugin struct {
	tracker *deployment_domain.DeploymentTracker
	logger  *slog.Logger
}

// NewDeploymentServerPlugin creates a new deployment server plugin
func NewDeploymentServerPlugin(
	tracker *deployment_domain.DeploymentTracker,
	logger *slog.Logger,
) domain.ServerPlugin {
	return &DeploymentServerPlugin{
		tracker: tracker,
		logger:  logger,
	}
}

// ServerPlugin interface implementation
func (p *DeploymentServerPlugin) ID() string   { return "deployment" }
func (p *DeploymentServerPlugin) Name() string { return "Dokku Deployment" }

func (p *DeploymentServerPlugin) Description() string {
	return "Dokku application deployment tracking and management"
}

func (p *DeploymentServerPlugin) Version() string { return "0.3.0" }

// No specific Dokku plugin dependency
func (p *DeploymentServerPlugin) DokkuPluginName() string { return "" }

// ResourceProvider implementation
func (p *DeploymentServerPlugin) GetResources(ctx context.Context) ([]domain.Resource, error) {
	deploymentResources, err := p.getDeploymentResources()
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment resources: %w", err)
	}

	buildLogResources, err := p.getBuildLogResources()
	if err != nil {
		return nil, fmt.Errorf("failed to get build log resources: %w", err)
	}

	// Combine all resources
	resources := append(deploymentResources, buildLogResources...)

	return resources, nil
}

// Get deployment resources (existing deployments)
func (p *DeploymentServerPlugin) getDeploymentResources() ([]domain.Resource, error) {
	deployments := p.tracker.GetAll()

	resources := make([]domain.Resource, 0, len(deployments))

	for _, deployment := range deployments {
		resources = append(resources, domain.Resource{
			URI:         fmt.Sprintf("dokku://deployment/%s", deployment.ID()),
			Name:        fmt.Sprintf("Deployment: %s", deployment.AppName()),
			Description: fmt.Sprintf("Status: %s", deployment.Status()),
			MIMEType:    "application/json",
			Handler:     p.handleDeploymentResource,
		})
	}

	return resources, nil
}

// Get build log resources
func (p *DeploymentServerPlugin) getBuildLogResources() ([]domain.Resource, error) {
	deployments := p.tracker.GetAll()

	resources := make([]domain.Resource, 0, len(deployments))

	for _, deployment := range deployments {
		// Only expose build logs for deployments that have logs
		if deployment.BuildLogs() != "" {
			resources = append(resources, domain.Resource{
				URI:         fmt.Sprintf("dokku://deployment/%s/logs", deployment.ID()),
				Name:        fmt.Sprintf("Build Logs: %s", deployment.AppName()),
				Description: fmt.Sprintf("Build logs for deployment %s", deployment.ID()),
				MIMEType:    "text/plain",
				Handler:     p.handleBuildLogsResource,
			})
		}
	}

	return resources, nil
}

// ToolProvider implementation
func (p *DeploymentServerPlugin) GetTools(ctx context.Context) ([]domain.Tool, error) {
	return []domain.Tool{
		{
			Name:        "get_deployment_status",
			Description: "Get the status and build log tail of a deployment",
			Builder:     buildGetDeploymentStatusTool,
			Handler:     p.handleGetDeploymentStatus,
		},
		{
			Name:        "list_deployments",
			Description: "List deployments tracked by this server",
			Builder:     buildListDeploymentsTool,
			Handler:     p.handleListDeployments,
		},
	}, nil
}

// maxBuildLogTail bounds how much build output a status call returns.
const maxBuildLogTail = 4000

// DeploymentView is the structured representation of a tracked deployment.
type DeploymentView struct {
	ID           string                             `json:"id" jsonschema:"Deployment identifier"`
	AppName      string                             `json:"app_name"`
	GitRef       string                             `json:"git_ref"`
	Status       deployment_domain.DeploymentStatus `json:"status" jsonschema:"One of pending, running, succeeded, failed, rolled_back"`
	Done         bool                               `json:"done" jsonschema:"True once the deployment reached a final status"`
	CreatedAt    time.Time                          `json:"created_at"`
	CompletedAt  *time.Time                         `json:"completed_at,omitempty"`
	Duration     string                             `json:"duration"`
	Error        string                             `json:"error,omitempty"`
	BuildLogTail string                             `json:"build_log_tail,omitempty" jsonschema:"Last part of the build output"`
}

// DeploymentList is the structured result of list_deployments.
type DeploymentList struct {
	Deployments []DeploymentView `json:"deployments"`
}

func newDeploymentView(d *deployment_domain.Deployment, withLogs bool) DeploymentView {
	view := DeploymentView{
		ID:          d.ID(),
		AppName:     d.AppName(),
		GitRef:      d.GitRef(),
		Status:      d.Status(),
		Done:        d.IsCompleted(),
		CreatedAt:   d.CreatedAt(),
		CompletedAt: d.CompletedAt(),
		Duration:    d.Duration().Round(time.Second).String(),
		Error:       d.ErrorMsg(),
	}
	if withLogs {
		logs := d.BuildLogs()
		if len(logs) > maxBuildLogTail {
			// The cut may fall inside a multi-byte character.
			logs = "..." + strings.ToValidUTF8(logs[len(logs)-maxBuildLogTail:], "")
		}
		view.BuildLogTail = logs
	}
	return view
}

func buildGetDeploymentStatusTool() mcp.Tool {
	return mcp.NewTool(
		"get_deployment_status",
		mcp.WithTitleAnnotation("Get deployment status"),
		mcp.WithDescription("Get the status of a deployment started with deploy_app. Poll until done is true. "+
			"Completed deployments are kept for a limited time."),
		mcp.WithString("deployment_id",
			mcp.Required(),
			mcp.Description("Deployment ID returned by deploy_app"),
		),
		mcp.WithOutputSchema[DeploymentView](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func buildListDeploymentsTool() mcp.Tool {
	return mcp.NewTool(
		"list_deployments",
		mcp.WithTitleAnnotation("List deployments"),
		mcp.WithDescription("List deployments tracked by this server, most recent first"),
		mcp.WithString("app_name",
			mcp.Description("Only list deployments of this application"),
		),
		mcp.WithBoolean("active_only",
			mcp.Description("Only list deployments that are still running"),
		),
		mcp.WithOutputSchema[DeploymentList](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

// deploymentIDPattern matches the IDs the tracker generates.
var deploymentIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}$`)

func validDeploymentID(id string) bool {
	return deploymentIDPattern.MatchString(id)
}

func (p *DeploymentServerPlugin) handleGetDeploymentStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	deploymentID, err := req.RequireString("deployment_id")
	if err != nil {
		return mcp.NewToolResultError("deployment_id is required"), nil
	}

	if !validDeploymentID(deploymentID) {
		return mcp.NewToolResultError("invalid deployment_id"), nil
	}

	deployment, err := p.tracker.GetByID(deploymentID)
	if err != nil {
		p.logger.Debug("Deployment status requested for unknown deployment", "deployment_id", deploymentID, "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Deployment '%s' not found. It may have expired; use get_app_status to check the application", deploymentID)), nil
	}

	return mcp.NewToolResultStructuredOnly(newDeploymentView(deployment, true)), nil
}

func (p *DeploymentServerPlugin) handleListDeployments(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appName := req.GetString("app_name", "")
	if appName != "" {
		name, err := appdomain.NewApplicationName(appName)
		if err != nil {
			return mcp.NewToolResultError("invalid app_name"), nil
		}
		appName = name.Value()
	}

	var deployments []*deployment_domain.Deployment
	if req.GetBool("active_only", false) {
		deployments = p.tracker.GetActive()
	} else {
		deployments = p.tracker.GetAll()
	}

	result := DeploymentList{Deployments: []DeploymentView{}}
	for _, d := range deployments {
		if appName != "" && d.AppName() != appName {
			continue
		}
		result.Deployments = append(result.Deployments, newDeploymentView(d, false))
	}
	slices.SortFunc(result.Deployments, func(a, b DeploymentView) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})

	return mcp.NewToolResultStructuredOnly(result), nil
}

// PromptProvider implementation
func (p *DeploymentServerPlugin) GetPrompts(ctx context.Context) ([]domain.Prompt, error) {
	// No prompts for now
	return []domain.Prompt{}, nil
}

// Resource handlers
func (p *DeploymentServerPlugin) handleDeploymentResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Parse URI to get deployment ID
	uri := req.Params.URI
	if !strings.HasPrefix(uri, "dokku://deployment/") {
		return nil, fmt.Errorf("invalid deployment resource URI: %s", uri)
	}

	// Extract deployment ID
	parts := strings.Split(strings.TrimPrefix(uri, "dokku://deployment/"), "/")
	deploymentID := parts[0]

	// Validate deployment ID for security
	if deploymentID == "" || len(deploymentID) > 100 || strings.ContainsAny(deploymentID, "\t\n\r\x00") {
		return nil, fmt.Errorf("invalid deployment ID format")
	}

	// Get deployment from tracker
	deployment, err := p.tracker.GetByID(deploymentID)
	if err != nil {
		p.logger.Error("deployment not found", "deployment_id", deploymentID, "error", err)
		return nil, fmt.Errorf("deployment not found")
	}

	// Define typed struct for deployment response
	type deploymentResourceResponse struct {
		ID           string     `json:"id"`
		AppName      string     `json:"app_name"`
		GitRef       string     `json:"git_ref"`
		Status       string     `json:"status"`
		CreatedAt    time.Time  `json:"created_at"`
		StartedAt    *time.Time `json:"started_at,omitempty"`
		CompletedAt  *time.Time `json:"completed_at,omitempty"`
		ErrorMsg     string     `json:"error_msg"`
		Duration     string     `json:"duration"`
		HasBuildLogs bool       `json:"has_build_logs"`
		BuildLogsURI string     `json:"build_logs_uri,omitempty"`
	}

	// Create typed deployment response
	response := deploymentResourceResponse{
		ID:           deployment.ID(),
		AppName:      deployment.AppName(),
		GitRef:       deployment.GitRef(),
		Status:       string(deployment.Status()),
		CreatedAt:    deployment.CreatedAt(),
		StartedAt:    deployment.StartedAt(),
		CompletedAt:  deployment.CompletedAt(),
		ErrorMsg:     deployment.ErrorMsg(),
		Duration:     deployment.Duration().String(),
		HasBuildLogs: deployment.BuildLogs() != "",
	}

	if deployment.BuildLogs() != "" {
		response.BuildLogsURI = fmt.Sprintf("dokku://deployment/%s/logs", deployment.ID())
	}

	// Serialize to JSON
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		p.logger.Error("failed to serialize deployment response", "error", err)
		return nil, fmt.Errorf("failed to serialize deployment info")
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

// Handle build logs resource
func (p *DeploymentServerPlugin) handleBuildLogsResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Parse URI to get deployment ID
	uri := req.Params.URI
	if !strings.HasPrefix(uri, "dokku://deployment/") {
		return nil, fmt.Errorf("invalid build logs resource URI: %s", uri)
	}

	// Extract deployment ID and verify it's a logs request
	parts := strings.Split(strings.TrimPrefix(uri, "dokku://deployment/"), "/")
	if len(parts) < 2 || parts[1] != "logs" {
		return nil, fmt.Errorf("invalid build logs resource URI format: %s", uri)
	}

	deploymentID := parts[0]

	// Get deployment from tracker
	deployment, err := p.tracker.GetByID(deploymentID)
	if err != nil {
		return nil, fmt.Errorf("deployment not found: %w", err)
	}

	// Get build logs
	buildLogs := deployment.BuildLogs()
	if buildLogs == "" {
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "text/plain",
				Text:     "No build logs available for this deployment.",
			},
		}, nil
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "text/plain",
			Text:     buildLogs,
		},
	}, nil
}

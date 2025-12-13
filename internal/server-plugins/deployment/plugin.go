package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

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

func (p *DeploymentServerPlugin) Version() string { return "0.1.0" }

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
	// No tools for now - deployment is handled via the apps plugin
	return []domain.Tool{}, nil
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

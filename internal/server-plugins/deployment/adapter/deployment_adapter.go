package adapter

import (
	"context"

	deployment_domain "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/deployment/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/shared"
)

// DeploymentServiceAdapter adapts the deployment plugin's service to the shared interface
// This allows other plugins to use deployment functionality without direct coupling
type DeploymentServiceAdapter struct {
	deploymentService deployment_domain.DeploymentService
}

// NewDeploymentServiceAdapter creates a new adapter instance
func NewDeploymentServiceAdapter(deploymentService deployment_domain.DeploymentService) shared.DeploymentService {
	return &DeploymentServiceAdapter{
		deploymentService: deploymentService,
	}
}

// Deploy implements the shared DeploymentService interface
func (a *DeploymentServiceAdapter) Deploy(ctx context.Context, appName string, options shared.DeployOptions) (*shared.DeploymentResult, error) {
	pluginOptions := deployment_domain.DeployOptions{
		RepoURL:   options.RepoURL,
		GitRef:    options.GitRef,
		BuildPack: options.Buildpack,
	}

	// Call the plugin's deployment service
	deployment, err := a.deploymentService.Deploy(ctx, appName, pluginOptions)
	if err != nil {
		return nil, err
	}

	// Convert plugin result to shared result
	return &shared.DeploymentResult{
		ID:          deployment.ID(),
		AppName:     deployment.AppName(),
		GitRef:      deployment.GitRef(),
		Status:      convertStatus(deployment.Status()),
		CreatedAt:   deployment.CreatedAt(),
		CompletedAt: deployment.CompletedAt(),
		ErrorMsg:    deployment.ErrorMsg(),
	}, nil
}

// Rollback implements the shared DeploymentService interface
func (a *DeploymentServiceAdapter) Rollback(ctx context.Context, appName string, version string) error {
	return a.deploymentService.Rollback(ctx, appName, version)
}

// GetHistory implements the shared DeploymentService interface
func (a *DeploymentServiceAdapter) GetHistory(ctx context.Context, appName string) ([]shared.DeploymentSummary, error) {
	deployments, err := a.deploymentService.GetHistory(ctx, appName)
	if err != nil {
		return nil, err
	}

	// Convert to shared summaries
	summaries := make([]shared.DeploymentSummary, len(deployments))
	for i, deployment := range deployments {
		summaries[i] = shared.DeploymentSummary{
			ID:        deployment.ID(),
			GitRef:    deployment.GitRef(),
			Status:    convertStatus(deployment.Status()),
			CreatedAt: deployment.CreatedAt(),
			Duration:  deployment.Duration(),
		}
	}

	return summaries, nil
}

// GetStatus implements the shared DeploymentService interface
func (a *DeploymentServiceAdapter) GetStatus(ctx context.Context, deploymentID string) (*shared.DeploymentResult, error) {
	deployment, err := a.deploymentService.GetByID(ctx, deploymentID)
	if err != nil {
		return nil, err
	}

	return &shared.DeploymentResult{
		ID:          deployment.ID(),
		AppName:     deployment.AppName(),
		GitRef:      deployment.GitRef(),
		Status:      convertStatus(deployment.Status()),
		CreatedAt:   deployment.CreatedAt(),
		CompletedAt: deployment.CompletedAt(),
		ErrorMsg:    deployment.ErrorMsg(),
	}, nil
}

// Cancel implements the shared DeploymentService interface
func (a *DeploymentServiceAdapter) Cancel(ctx context.Context, deploymentID string) error {
	return a.deploymentService.Cancel(ctx, deploymentID)
}

// convertStatus converts plugin-specific status to shared status
func convertStatus(pluginStatus deployment_domain.DeploymentStatus) shared.DeploymentStatus {
	switch pluginStatus {
	case deployment_domain.DeploymentStatusPending:
		return shared.DeploymentStatusPending
	case deployment_domain.DeploymentStatusRunning:
		return shared.DeploymentStatusRunning
	case deployment_domain.DeploymentStatusSucceeded:
		return shared.DeploymentStatusSucceeded
	case deployment_domain.DeploymentStatusFailed:
		return shared.DeploymentStatusFailed
	case deployment_domain.DeploymentStatusRolledBack:
		return shared.DeploymentStatusRolledBack
	default:
		return shared.DeploymentStatusFailed
	}
}

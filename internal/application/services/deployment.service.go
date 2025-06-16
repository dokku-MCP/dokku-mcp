package services

import (
	"context"

	"github.com/alex-galey/dokku-mcp/internal/domain/dokku/application"
	"github.com/alex-galey/dokku-mcp/internal/domain/dokku/deployment"
	deployment_services "github.com/alex-galey/dokku-mcp/internal/domain/dokku/deployment/services"
)

// ApplicationDeploymentService provides application-layer orchestration for deployment operations
// This service coordinates between domain services and provides the interface for the main function
type ApplicationDeploymentService interface {
	DeployApplication(ctx context.Context, appName string, options DeploymentOptions) (*deployment.Deployment, error)
	GetDeploymentStatus(ctx context.Context, appName string) (*deployment.Deployment, error)
	ListDeployments(ctx context.Context, appName string) ([]*deployment.Deployment, error)
}

// DeploymentOptions contains options for application deployment
type DeploymentOptions struct {
	GitRef     string
	BuildImage string
	RunImage   string
	ForceClean bool
	NoCache    bool
}

// applicationDeploymentService implements ApplicationDeploymentService
type applicationDeploymentService struct {
	appRepo           application.ApplicationRepository
	deploymentService deployment_services.DeploymentService
}

// NewApplicationDeploymentService creates a new application deployment service
func NewApplicationDeploymentService(
	appRepo application.ApplicationRepository,
	deploymentService deployment_services.DeploymentService,
) ApplicationDeploymentService {
	return &applicationDeploymentService{
		appRepo:           appRepo,
		deploymentService: deploymentService,
	}
}

// DeployApplication orchestrates the deployment of an application
func (s *applicationDeploymentService) DeployApplication(ctx context.Context, appName string, options DeploymentOptions) (*deployment.Deployment, error) {
	// Get application from repository
	appNameVO, err := application.NewApplicationName(appName)
	if err != nil {
		return nil, err
	}

	app, err := s.appRepo.GetByName(ctx, appNameVO)
	if err != nil {
		return nil, err
	}

	// Create git reference if provided
	var gitRef *application.GitRef
	if options.GitRef != "" {
		gitRef, err = application.NewGitRef(options.GitRef)
		if err != nil {
			return nil, err
		}
	}

	// Prepare deployment options
	deployOpts := &application.DeploymentOptions{
		BuildImage: options.BuildImage,
		RunImage:   options.RunImage,
		ForceClean: options.ForceClean,
		NoCache:    options.NoCache,
	}

	// Deploy through domain service
	if gitRef != nil {
		if err := app.Deploy(gitRef, deployOpts); err != nil {
			return nil, err
		}
	}

	// Save updated application state
	if err := s.appRepo.Save(ctx, app); err != nil {
		return nil, err
	}

	// Use deployment service to handle infrastructure deployment
	return s.deploymentService.Deploy(ctx, appName, deployment_services.DeployOptions{
		GitRef:    options.GitRef,
		BuildPack: options.BuildImage, // Map BuildImage to BuildPack for now
	})
}

// GetDeploymentStatus gets the current deployment status
func (s *applicationDeploymentService) GetDeploymentStatus(ctx context.Context, appName string) (*deployment.Deployment, error) {
	// For now, get the latest deployment from history
	deployments, err := s.deploymentService.GetHistory(ctx, appName)
	if err != nil {
		return nil, err
	}

	if len(deployments) == 0 {
		return nil, deployment.ErrDeploymentNotFound
	}

	return deployments[0], nil // Return the most recent deployment
}

// ListDeployments lists all deployments for an application
func (s *applicationDeploymentService) ListDeployments(ctx context.Context, appName string) ([]*deployment.Deployment, error) {
	return s.deploymentService.GetHistory(ctx, appName)
}

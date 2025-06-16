package dokku

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	deployment_domain "github.com/alex-galey/dokku-mcp/internal/domain/dokku/deployment"
	deployment_services "github.com/alex-galey/dokku-mcp/internal/domain/dokku/deployment/services"
)

// deploymentInfrastructure implements the simplified DeploymentInfrastructure interface
// Handles only external system calls (Dokku commands) - NO BUSINESS LOGIC
type deploymentInfrastructure struct {
	client DokkuClient
	logger *slog.Logger
}

// NewDeploymentInfrastructure creates a new deployment infrastructure implementation
func NewDeploymentInfrastructure(client DokkuClient, logger *slog.Logger) deployment_services.DeploymentInfrastructure {
	return &deploymentInfrastructure{
		client: client,
		logger: logger,
	}
}

// CheckApplicationExists checks if application exists in Dokku - INFRASTRUCTURE ONLY
func (s *deploymentInfrastructure) CheckApplicationExists(ctx context.Context, appName string) (bool, error) {
	_, err := s.client.ExecuteCommand(ctx, "apps:exists", []string{appName})
	if err != nil {
		return false, nil
	}
	return true, nil
}

// CreateApplication creates application in Dokku - INFRASTRUCTURE ONLY
func (s *deploymentInfrastructure) CreateApplication(ctx context.Context, appName string) error {
	_, err := s.client.ExecuteCommand(ctx, "apps:create", []string{appName})
	if err != nil {
		return fmt.Errorf("failed to create application in Dokku: %w", err)
	}
	return nil
}

// SetBuildpack sets buildpack for application in Dokku - INFRASTRUCTURE ONLY
func (s *deploymentInfrastructure) SetBuildpack(ctx context.Context, appName string, buildpack string) error {
	_, err := s.client.ExecuteCommand(ctx, "buildpacks:set", []string{appName, buildpack})
	if err != nil {
		return fmt.Errorf("failed to set buildpack in Dokku: %w", err)
	}
	return nil
}

// PerformGitDeploy executes git deployment in Dokku - INFRASTRUCTURE ONLY
func (s *deploymentInfrastructure) PerformGitDeploy(ctx context.Context, appName string, gitRef string) error {
	args := []string{appName}
	if gitRef != "" {
		args = append(args, gitRef)
	}

	_, err := s.client.ExecuteCommand(ctx, "git:sync", args)
	if err != nil {
		return fmt.Errorf("failed to perform git deploy in Dokku: %w", err)
	}

	return nil
}

// ParseDeploymentHistory retrieves deployment history from Dokku - INFRASTRUCTURE ONLY
func (s *deploymentInfrastructure) ParseDeploymentHistory(ctx context.Context, appName string) ([]*deployment_domain.Deployment, error) {
	// Get events from Dokku
	eventsOutput, err := s.client.ExecuteCommand(ctx, "events", []string{appName})
	if err != nil {
		return nil, fmt.Errorf("failed to get events from Dokku: %w", err)
	}

	// Parse the events output to extract deployments
	deployments := s.parseEventsOutput(string(eventsOutput), appName)

	// Sort by creation time (most recent first)
	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].CreatedAt().After(deployments[j].CreatedAt())
	})

	return deployments, nil
}

// parseEventsOutput parses Dokku events output to extract deployments - INFRASTRUCTURE PARSING
func (s *deploymentInfrastructure) parseEventsOutput(eventsOutput, appName string) []*deployment_domain.Deployment {
	lines := strings.Split(eventsOutput, "\n")
	var deployments []*deployment_domain.Deployment

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "deploy") {
			continue
		}

		deployment := s.parseEventLine(line, appName)
		if deployment != nil {
			deployments = append(deployments, deployment)
		}
	}

	return deployments
}

// parseEventLine parses a single event line to create deployment - INFRASTRUCTURE PARSING
func (s *deploymentInfrastructure) parseEventLine(line, appName string) *deployment_domain.Deployment {
	// Extract git ref if available, otherwise use "main"
	gitRef := "main"
	parts := strings.Fields(line)
	for _, part := range parts {
		if strings.Contains(part, ":") && strings.Contains(part, "git") {
			gitParts := strings.Split(part, ":")
			if len(gitParts) > 1 {
				gitRef = gitParts[1]
			}
		}
	}

	// Create deployment entity
	deployment, err := deployment_domain.NewDeployment(appName, gitRef)
	if err != nil {
		s.logger.Warn("Failed to create deployment from event", "error", err)
		return nil
	}

	// Set complete status
	deployment.Complete()

	return deployment
}

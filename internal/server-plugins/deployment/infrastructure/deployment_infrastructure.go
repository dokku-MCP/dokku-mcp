package dokku

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	dokku_client "github.com/alex-galey/dokku-mcp/internal/dokku-api"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/deployment/domain"
)

// deploymentInfrastructure implements the simplified DeploymentInfrastructure interface
// Handles only external system calls (Dokku commands) - NO BUSINESS LOGIC
type deploymentInfrastructure struct {
	client dokku_client.DokkuClient
	logger *slog.Logger

	// Deployment locking to prevent concurrent deployments of the same app
	deploymentMutex   sync.Mutex
	activeDeployments map[string]bool
}

// NewDeploymentInfrastructure creates a new deployment infrastructure implementation
func NewDeploymentInfrastructure(client dokku_client.DokkuClient, logger *slog.Logger) domain.DeploymentInfrastructure {
	return &deploymentInfrastructure{
		client:            client,
		logger:            logger,
		activeDeployments: make(map[string]bool),
	}
}

// executeCommand wraps the client's ExecuteCommand with deployment-specific context and validation
func (s *deploymentInfrastructure) executeCommand(ctx context.Context, command domain.DeploymentCommand, args []string) ([]byte, error) {
	if !command.IsValid() {
		return nil, fmt.Errorf("invalid deployment command: %s", command)
	}

	return s.client.ExecuteCommand(ctx, command.String(), args)
}

// SetBuildpack sets buildpack for application in Dokku - INFRASTRUCTURE ONLY
func (s *deploymentInfrastructure) SetBuildpack(ctx context.Context, appName string, buildpack string) error {
	_, err := s.executeCommand(ctx, domain.CommandBuildpacksSet, []string{appName, buildpack})
	if err != nil {
		return fmt.Errorf("failed to set buildpack in Dokku: %w", err)
	}
	return nil
}

// PerformGitDeploy executes git deployment in Dokku - INFRASTRUCTURE ONLY
func (s *deploymentInfrastructure) PerformGitDeploy(ctx context.Context, appName, repoURL, gitRef string) error {
	s.logger.Debug("Performing git deployment", "app_name", appName, "repo_url", repoURL, "git_ref", gitRef)

	// Check for concurrent deployment
	s.deploymentMutex.Lock()
	if s.activeDeployments[appName] {
		s.deploymentMutex.Unlock()
		return fmt.Errorf("deployment already in progress for application %s", appName)
	}
	s.activeDeployments[appName] = true
	s.deploymentMutex.Unlock()

	// Ensure cleanup on exit
	defer func() {
		s.deploymentMutex.Lock()
		delete(s.activeDeployments, appName)
		s.deploymentMutex.Unlock()
		s.logger.Debug("Deployment lock released", "app_name", appName)
	}()

	// Perform git sync (fast operation)
	_, err := s.executeCommand(ctx, domain.CommandGitSync, []string{appName, repoURL, gitRef})
	if err != nil {
		return fmt.Errorf("git sync failed: %w", err)
	}

	s.logger.Debug("Git sync completed, triggering async rebuild", "app_name", appName)

	// Trigger async rebuild (this will happen in background)
	s.performAsyncRebuild(appName, gitRef)

	return nil
}

// performAsyncRebuild performs the rebuild operation in the background
func (s *deploymentInfrastructure) performAsyncRebuild(appName, gitRef string) {
	s.logger.Info("Starting async rebuild", "app_name", appName, "git_ref", gitRef)

	// Use a truly fire-and-forget approach that doesn't wait for the command to complete
	go func() {
		// Create a background context with extended timeout for the build process
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		s.logger.Debug("Executing async ps:rebuild command", "app_name", appName)

		// Execute the rebuild command - this may timeout but the Dokku process will continue
		_, err := s.executeCommand(ctx, domain.CommandPsRebuild, []string{appName})

		if err != nil {
			// SSH timeouts are expected for long builds - log but don't treat as error
			if strings.Contains(err.Error(), "signal: killed") ||
				strings.Contains(err.Error(), "context deadline exceeded") ||
				strings.Contains(err.Error(), "connection closed") ||
				strings.Contains(err.Error(), "timeout") {
				s.logger.Info("Async rebuild command sent successfully - build continuing in background",
					"app_name", appName, "git_ref", gitRef,
					"note", "SSH connection closed as expected for long builds")
			} else {
				s.logger.Error("Async rebuild failed with unexpected error",
					"app_name", appName, "error", err)
			}
		} else {
			s.logger.Info("Async rebuild completed successfully",
				"app_name", appName, "git_ref", gitRef)
		}
	}()

	s.logger.Info("Async rebuild command dispatched", "app_name", appName, "git_ref", gitRef)
}

// ParseDeploymentHistory retrieves deployment history from Dokku - INFRASTRUCTURE ONLY
func (s *deploymentInfrastructure) ParseDeploymentHistory(ctx context.Context, appName string) ([]*domain.Deployment, error) {
	// Get events from Dokku
	eventsOutput, err := s.executeCommand(ctx, domain.CommandEvents, []string{appName})
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
func (s *deploymentInfrastructure) parseEventsOutput(eventsOutput, appName string) []*domain.Deployment {
	lines := strings.Split(eventsOutput, "\n")
	var deployments []*domain.Deployment

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
func (s *deploymentInfrastructure) parseEventLine(line, appName string) *domain.Deployment {
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
	deployment, err := domain.NewDeployment(appName, gitRef)
	if err != nil {
		s.logger.Warn("Failed to create deployment from event", "error", err)
		return nil
	}

	// Set complete status
	deployment.Complete()

	return deployment
}

package dokku

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	dokku_client "github.com/dokku-mcp/dokku-mcp/internal/dokku-api"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/deployment/domain"
)

// deploymentInfrastructure implements the simplified DeploymentInfrastructure interface
// Handles only external system calls (Dokku commands) - NO BUSINESS LOGIC
type deploymentInfrastructure struct {
	client dokku_client.DokkuClient
	logger *slog.Logger

	// Deployment tracking
	tracker *domain.DeploymentTracker
	poller  *domain.DeploymentPoller

	// Deployment locking to prevent concurrent deployments of the same app
	deploymentMutex   sync.Mutex
	activeDeployments map[string]bool
}

// NewDeploymentInfrastructure creates a new deployment infrastructure implementation
func NewDeploymentInfrastructure(
	client dokku_client.DokkuClient,
	logger *slog.Logger,
	tracker *domain.DeploymentTracker,
	poller *domain.DeploymentPoller,
) domain.DeploymentInfrastructure {
	return &deploymentInfrastructure{
		client:            client,
		logger:            logger,
		tracker:           tracker,
		poller:            poller,
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
func (s *deploymentInfrastructure) PerformGitDeploy(ctx context.Context, deploymentID, appName, repoURL, gitRef string) error {
	s.logger.Debug("Performing git deployment",
		"deployment_id", deploymentID,
		"app_name", appName,
		"repo_url", repoURL,
		"git_ref", gitRef)

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
		s.logger.Debug("Deployment lock released", "app_name", appName, "deployment_id", deploymentID)
	}()

	// Perform git sync. Some environments may need a slightly longer timeout
	// than the default client timeout due to network and repository size.
	gitSyncCtx := ctx
	if _, ok := ctx.Deadline(); !ok {
		// Provide a modest 2 minute timeout for git:sync when caller didn't set one
		var cancel context.CancelFunc
		gitSyncCtx, cancel = context.WithTimeout(ctx, 2*time.Minute)
		defer cancel()
	}
	_, err := s.executeCommand(gitSyncCtx, domain.CommandGitSync, []string{appName, repoURL, gitRef})
	if err != nil {
		return fmt.Errorf("git sync failed: %w", err)
	}

	s.logger.Debug("Git sync completed, triggering async rebuild",
		"app_name", appName,
		"deployment_id", deploymentID)

	// Trigger async rebuild with tracking
	s.performAsyncRebuild(deploymentID, appName, gitRef)

	return nil
}

// performAsyncRebuild performs the rebuild operation with proper tracking
func (s *deploymentInfrastructure) performAsyncRebuild(deploymentID, appName, gitRef string) {
	s.logger.Info("Starting tracked async rebuild",
		"deployment_id", deploymentID,
		"app_name", appName,
		"git_ref", gitRef)

	// Start polling for status in background
	if s.poller != nil {
		s.poller.StartPolling(context.Background(), deploymentID, appName)
	}

	// Trigger the rebuild command (may timeout but build continues on Dokku)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		s.logger.Debug("Executing ps:rebuild command", "deployment_id", deploymentID, "app_name", appName)

		_, err := s.executeCommand(ctx, domain.CommandPsRebuild, []string{appName})

		// SSH timeout is expected - the poller will track actual status
		if err != nil {
			if dokku_client.IsNotFoundError(err) {
				s.logger.Warn("Rebuild command skipped (app missing)",
					"deployment_id", deploymentID,
					"app_name", appName)
				// Update tracker with failed status but without surfacing an error
				if s.tracker != nil {
					_ = s.tracker.UpdateStatus(deploymentID, domain.DeploymentStatusFailed, "application no longer exists")
				}
			} else if strings.Contains(err.Error(), "signal: killed") ||
				strings.Contains(err.Error(), "context deadline exceeded") ||
				strings.Contains(err.Error(), "connection closed") ||
				strings.Contains(err.Error(), "timeout") {
				s.logger.Info("Rebuild command sent, SSH connection closed (expected for long builds)",
					"deployment_id", deploymentID,
					"app_name", appName,
					"note", "Poller will track actual completion status")
			} else {
				// Demote expected not-found races using sentinel classification only
				if dokku_client.IsNotFoundError(err) {
					s.logger.Warn("Rebuild aborted (app removed during deploy)",
						"deployment_id", deploymentID,
						"app_name", appName)
					if s.tracker != nil {
						_ = s.tracker.UpdateStatus(deploymentID, domain.DeploymentStatusFailed, "application no longer exists")
					}
				} else {
					s.logger.Error("Rebuild command failed",
						"deployment_id", deploymentID,
						"app_name", appName,
						"error", err)

					// Update tracker with error
					if s.tracker != nil {
						_ = s.tracker.UpdateStatus(deploymentID, domain.DeploymentStatusFailed, err.Error())
					}
				}
			}
		}
	}()
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

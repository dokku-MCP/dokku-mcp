package dokku

import (
	"context"
	"fmt"
	"strings"

	dokku_client "github.com/alex-galey/dokku-mcp/internal/dokku-api"
	appdomain "github.com/alex-galey/dokku-mcp/internal/server-plugins/app/domain"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/deployment/domain"
)

// deploymentStatusChecker implements DeploymentStatusChecker
type deploymentStatusChecker struct {
	client dokku_client.DokkuClient
}

// NewDeploymentStatusChecker creates a new status checker
func NewDeploymentStatusChecker(client dokku_client.DokkuClient) domain.DeploymentStatusChecker {
	return &deploymentStatusChecker{
		client: client,
	}
}

// CheckStatus checks the deployment status by querying Dokku
func (dsc *deploymentStatusChecker) CheckStatus(ctx context.Context, appName string) (domain.DeploymentStatus, string, error) {
	if err := validateAppName(appName); err != nil {
		return domain.DeploymentStatusFailed, "", err
	}
	// First, check if app exists and get its report
	reportOutput, err := dsc.client.ExecuteCommand(ctx, "apps:report", []string{appName})
	if err != nil {
		// If app no longer exists, treat as a terminal state without surfacing an error
		if dokku_client.IsNotFoundError(err) {
			return domain.DeploymentStatusFailed, "application no longer exists", nil
		}
		// Otherwise, keep previous behavior (transient pending)
		return domain.DeploymentStatusPending, "", fmt.Errorf("failed to get app report: %w", err)
	}

	reportStr := string(reportOutput)
	lowerReport := strings.ToLower(reportStr)

	// Check if app is deployed by looking for "deployed: true"
	if !strings.Contains(reportStr, "deployed: true") && !strings.Contains(reportStr, "Deployed:") {
		// App exists but not yet deployed – treat as pending to avoid early log fetching
		return domain.DeploymentStatusPending, "", nil
	}

	if strings.Contains(lowerReport, "app locked: true") {
		return domain.DeploymentStatusFailed, "application locked after deployment", nil
	}

	// Check process status
	psOutput, err := dsc.client.ExecuteCommand(ctx, "ps:report", []string{appName})
	if err != nil {
		return domain.DeploymentStatusRunning, "unable to retrieve process report", nil
	}

	psStr := string(psOutput)

	// Check for running containers
	if strings.Contains(psStr, "Processes running") || strings.Contains(psStr, "running:") {
		// Look for actual running processes
		if strings.Contains(psStr, "running: 0") || strings.Contains(psStr, "Processes running: 0") {
			if strings.Contains(lowerReport, "app locked: true") {
				return domain.DeploymentStatusFailed, "application locked after deployment", nil
			}
			return domain.DeploymentStatusPending, "waiting for processes to start", nil
		}
		// Processes are running - deployment succeeded
		return domain.DeploymentStatusSucceeded, "", nil
	}

	// Check for specific failure indicators in report
	if strings.Contains(reportStr, "deploy source:") {
		// Has a deploy source, consider it deployed
		return domain.DeploymentStatusSucceeded, "", nil
	}

	// Default to running if we can't determine
	return domain.DeploymentStatusRunning, "", nil
}

// GetLogs retrieves recent logs from the application
func (dsc *deploymentStatusChecker) GetLogs(ctx context.Context, appName string, lines int) (string, error) {
	if err := validateAppName(appName); err != nil {
		return "", err
	}
	// Try to get logs with tail
	output, err := dsc.client.ExecuteCommand(ctx, "logs", []string{appName, "--num", fmt.Sprintf("%d", lines)})
	if err != nil {
		// Logs might not be available yet
		return "", fmt.Errorf("failed to get logs: %w", err)
	}

	return string(output), nil
}

func validateAppName(appName string) error {
	if _, err := appdomain.NewApplicationName(appName); err != nil {
		return fmt.Errorf("invalid app name: %w", err)
	}
	return nil
}

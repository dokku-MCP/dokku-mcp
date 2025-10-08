package dokku

import (
	"context"
	"fmt"
	"strings"

	dokku_client "github.com/alex-galey/dokku-mcp/internal/dokku-api"
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
	// First, check if app exists and get its report
	reportOutput, err := dsc.client.ExecuteCommand(ctx, "apps:report", []string{appName})
	if err != nil {
		// App might not exist yet or connection failed
		return domain.DeploymentStatusPending, "", fmt.Errorf("failed to get app report: %w", err)
	}

	reportStr := string(reportOutput)

	// Check if app is deployed by looking for "deployed: true"
	if !strings.Contains(reportStr, "deployed: true") && !strings.Contains(reportStr, "Deployed:") {
		// App exists but not yet deployed
		return domain.DeploymentStatusRunning, "", nil
	}

	// Check process status
	psOutput, err := dsc.client.ExecuteCommand(ctx, "ps:report", []string{appName})
	if err != nil {
		// Can't determine process status, assume still running
		return domain.DeploymentStatusRunning, "", nil
	}

	psStr := string(psOutput)

	// Check for running containers
	if strings.Contains(psStr, "Processes running") || strings.Contains(psStr, "running:") {
		// Look for actual running processes
		if strings.Contains(psStr, "running: 0") || strings.Contains(psStr, "Processes running: 0") {
			// No processes running - check if deployment failed
			logs, _ := dsc.GetLogs(ctx, appName, 100)
			if strings.Contains(strings.ToLower(logs), "error") ||
				strings.Contains(strings.ToLower(logs), "failed") ||
				strings.Contains(strings.ToLower(logs), "exit code") {
				return domain.DeploymentStatusFailed, "deployment failed - check logs for details", nil
			}
			// Might be scaling down or just deployed
			return domain.DeploymentStatusSucceeded, "", nil
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
	// Try to get logs with tail
	output, err := dsc.client.ExecuteCommand(ctx, "logs", []string{appName, "--num", fmt.Sprintf("%d", lines)})
	if err != nil {
		// Logs might not be available yet
		return "", fmt.Errorf("failed to get logs: %w", err)
	}

	return string(output), nil
}

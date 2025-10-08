package domain

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// DeploymentStatusChecker interface for checking deployment status in Dokku
type DeploymentStatusChecker interface {
	CheckStatus(ctx context.Context, appName string) (DeploymentStatus, string, error)
	GetLogs(ctx context.Context, appName string, lines int) (string, error)
}

// DeploymentPoller polls Dokku for deployment status updates
type DeploymentPoller struct {
	tracker       *DeploymentTracker
	statusChecker DeploymentStatusChecker
	logger        *slog.Logger
	pollInterval  time.Duration
	maxPollTime   time.Duration
	stopChan      chan struct{}
	activePolls   map[string]context.CancelFunc
	pollMutex     sync.RWMutex
}

// NewDeploymentPoller creates a new deployment poller
func NewDeploymentPoller(
	tracker *DeploymentTracker,
	statusChecker DeploymentStatusChecker,
	logger *slog.Logger,
	pollInterval time.Duration,
	maxPollTime time.Duration,
) *DeploymentPoller {
	return &DeploymentPoller{
		tracker:       tracker,
		statusChecker: statusChecker,
		logger:        logger,
		pollInterval:  pollInterval,
		maxPollTime:   maxPollTime,
		stopChan:      make(chan struct{}),
		activePolls:   make(map[string]context.CancelFunc),
	}
}

// StartPolling begins polling for a deployment's status
func (dp *DeploymentPoller) StartPolling(ctx context.Context, deploymentID, appName string) {
	dp.logger.Info("Starting deployment polling",
		"deployment_id", deploymentID,
		"app_name", appName,
		"poll_interval", dp.pollInterval,
		"max_duration", dp.maxPollTime)

	// Create a context with timeout for the entire polling operation
	pollCtx, cancel := context.WithTimeout(ctx, dp.maxPollTime)

	// Store cancel function
	dp.pollMutex.Lock()
	dp.activePolls[deploymentID] = cancel
	dp.pollMutex.Unlock()

	// Start polling in goroutine
	go dp.pollDeploymentStatus(pollCtx, deploymentID, appName, cancel)
}

// pollDeploymentStatus polls Dokku for deployment status updates
func (dp *DeploymentPoller) pollDeploymentStatus(ctx context.Context, deploymentID, appName string, cancel context.CancelFunc) {
	defer func() {
		cancel()
		// Remove from active polls
		dp.pollMutex.Lock()
		delete(dp.activePolls, deploymentID)
		dp.pollMutex.Unlock()
	}()

	ticker := time.NewTicker(dp.pollInterval)
	defer ticker.Stop()

	consecutiveErrors := 0
	maxConsecutiveErrors := 5

	for {
		select {
		case <-ctx.Done():
			dp.logger.Warn("Deployment polling timeout or cancelled",
				"deployment_id", deploymentID,
				"app_name", appName,
				"error", ctx.Err())

			// Mark as failed if timeout
			if ctx.Err() == context.DeadlineExceeded {
				_ = dp.tracker.UpdateStatus(deploymentID, DeploymentStatusFailed,
					fmt.Sprintf("deployment timed out after %v", dp.maxPollTime))
			}
			return

		case <-dp.stopChan:
			dp.logger.Info("Deployment polling stopped",
				"deployment_id", deploymentID,
				"app_name", appName)
			return

		case <-ticker.C:
			status, errorMsg, err := dp.statusChecker.CheckStatus(ctx, appName)
			if err != nil {
				consecutiveErrors++
				dp.logger.Warn("Failed to check deployment status",
					"deployment_id", deploymentID,
					"app_name", appName,
					"error", err,
					"consecutive_errors", consecutiveErrors)

				// If too many consecutive errors, mark as failed
				if consecutiveErrors >= maxConsecutiveErrors {
					dp.logger.Error("Too many consecutive polling errors, marking deployment as failed",
						"deployment_id", deploymentID,
						"app_name", appName)
					_ = dp.tracker.UpdateStatus(deploymentID, DeploymentStatusFailed,
						fmt.Sprintf("status check failed after %d attempts: %v", consecutiveErrors, err))
					return
				}
				continue
			}

			// Reset error counter on success
			consecutiveErrors = 0

			// Get current deployment state
			deployment, err := dp.tracker.GetByID(deploymentID)
			if err != nil {
				dp.logger.Warn("Deployment not found in tracker",
					"deployment_id", deploymentID)
				return
			}

			previousStatus := deployment.Status()

			// Update status in tracker
			if err := dp.tracker.UpdateStatus(deploymentID, status, errorMsg); err != nil {
				dp.logger.Warn("Failed to update deployment status",
					"deployment_id", deploymentID,
					"error", err)
				continue
			}

			dp.logger.Debug("Deployment status checked",
				"deployment_id", deploymentID,
				"app_name", appName,
				"previous_status", previousStatus,
				"current_status", status)

			// Fetch and append logs if deployment is running
			if status == DeploymentStatusRunning || status == DeploymentStatusSucceeded || status == DeploymentStatusFailed {
				logs, err := dp.statusChecker.GetLogs(ctx, appName, 50)
				if err != nil {
					dp.logger.Debug("Failed to fetch logs",
						"deployment_id", deploymentID,
						"error", err)
				} else if logs != "" {
					_ = dp.tracker.AddLogs(deploymentID, logs)
				}
			}

			// Stop polling if deployment is completed
			if status == DeploymentStatusSucceeded || status == DeploymentStatusFailed {
				dp.logger.Info("Deployment completed",
					"deployment_id", deploymentID,
					"app_name", appName,
					"status", status,
					"error", errorMsg)
				return
			}
		}
	}
}

// StopPolling stops polling for a specific deployment
func (dp *DeploymentPoller) StopPolling(deploymentID string) {
	dp.pollMutex.Lock()
	defer dp.pollMutex.Unlock()

	if cancel, exists := dp.activePolls[deploymentID]; exists {
		cancel()
		delete(dp.activePolls, deploymentID)
		dp.logger.Info("Stopped polling for deployment", "deployment_id", deploymentID)
	}
}

// Shutdown stops all active polling
func (dp *DeploymentPoller) Shutdown() {
	close(dp.stopChan)

	dp.pollMutex.Lock()
	defer dp.pollMutex.Unlock()

	for deploymentID, cancel := range dp.activePolls {
		cancel()
		dp.logger.Info("Cancelled polling during shutdown", "deployment_id", deploymentID)
	}

	// Clear the map
	dp.activePolls = make(map[string]context.CancelFunc)
}

// GetActivePollCount returns the number of active polls
func (dp *DeploymentPoller) GetActivePollCount() int {
	dp.pollMutex.RLock()
	defer dp.pollMutex.RUnlock()

	return len(dp.activePolls)
}

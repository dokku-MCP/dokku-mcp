package domain

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DeploymentTracker manages in-memory state of active deployments
type DeploymentTracker struct {
	deployments map[string]*TrackedDeployment
	mu          sync.RWMutex
	cleanupTTL  time.Duration
}

// TrackedDeployment represents a deployment being tracked
type TrackedDeployment struct {
	Deployment  *Deployment
	StartedAt   time.Time
	LastChecked time.Time
	mu          sync.RWMutex
}

// NewDeploymentTracker creates a new deployment tracker
func NewDeploymentTracker() *DeploymentTracker {
	tracker := &DeploymentTracker{
		deployments: make(map[string]*TrackedDeployment),
		cleanupTTL:  5 * time.Minute, // Clean up completed deployments after 5 minutes
	}

	// Start cleanup goroutine
	go tracker.cleanupLoop()

	return tracker
}

// Track starts tracking a deployment
func (dt *DeploymentTracker) Track(deployment *Deployment) error {
	if deployment == nil {
		return fmt.Errorf("deployment cannot be nil")
	}

	tracked := &TrackedDeployment{
		Deployment:  deployment,
		StartedAt:   time.Now(),
		LastChecked: time.Now(),
	}

	dt.mu.Lock()
	dt.deployments[deployment.ID()] = tracked
	dt.mu.Unlock()

	return nil
}

// GetByID retrieves a tracked deployment by ID
func (dt *DeploymentTracker) GetByID(deploymentID string) (*Deployment, error) {
	dt.mu.RLock()
	tracked, exists := dt.deployments[deploymentID]
	dt.mu.RUnlock()

	if !exists {
		return nil, ErrDeploymentNotFound
	}

	tracked.mu.RLock()
	defer tracked.mu.RUnlock()

	return tracked.Deployment, nil
}

// UpdateStatus updates the status of a tracked deployment
func (dt *DeploymentTracker) UpdateStatus(deploymentID string, status DeploymentStatus, errorMsg string) error {
	dt.mu.RLock()
	tracked, exists := dt.deployments[deploymentID]
	dt.mu.RUnlock()

	if !exists {
		return ErrDeploymentNotFound
	}

	tracked.mu.Lock()
	defer tracked.mu.Unlock()

	tracked.LastChecked = time.Now()

	switch status {
	case DeploymentStatusRunning:
		if tracked.Deployment.Status() == DeploymentStatusPending {
			tracked.Deployment.Start()
		}
	case DeploymentStatusSucceeded:
		tracked.Deployment.Complete()
	case DeploymentStatusFailed:
		tracked.Deployment.Fail(errorMsg)
	}

	return nil
}

// AddLogs appends logs to a tracked deployment
func (dt *DeploymentTracker) AddLogs(deploymentID string, logs string) error {
	dt.mu.RLock()
	tracked, exists := dt.deployments[deploymentID]
	dt.mu.RUnlock()

	if !exists {
		return ErrDeploymentNotFound
	}

	tracked.mu.Lock()
	defer tracked.mu.Unlock()

	tracked.Deployment.AddBuildLogs(logs)
	return nil
}

// Remove removes a deployment from tracking
func (dt *DeploymentTracker) Remove(deploymentID string) {
	dt.mu.Lock()
	delete(dt.deployments, deploymentID)
	dt.mu.Unlock()
}

// GetAll returns all tracked deployments
func (dt *DeploymentTracker) GetAll() []*Deployment {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	deployments := make([]*Deployment, 0, len(dt.deployments))
	for _, tracked := range dt.deployments {
		tracked.mu.RLock()
		deployments = append(deployments, tracked.Deployment)
		tracked.mu.RUnlock()
	}

	return deployments
}

// GetActive returns all deployments that are not completed
func (dt *DeploymentTracker) GetActive() []*Deployment {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	active := make([]*Deployment, 0)
	for _, tracked := range dt.deployments {
		tracked.mu.RLock()
		if !tracked.Deployment.IsCompleted() {
			active = append(active, tracked.Deployment)
		}
		tracked.mu.RUnlock()
	}

	return active
}

// cleanupLoop periodically removes old completed deployments
func (dt *DeploymentTracker) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		dt.cleanup()
	}
}

// cleanup removes completed deployments older than TTL
func (dt *DeploymentTracker) cleanup() {
	now := time.Now()
	var toRemove []string

	dt.mu.RLock()
	for id, tracked := range dt.deployments {
		tracked.mu.RLock()
		if tracked.Deployment.IsCompleted() {
			age := now.Sub(tracked.LastChecked)
			if age > dt.cleanupTTL {
				toRemove = append(toRemove, id)
			}
		}
		tracked.mu.RUnlock()
	}
	dt.mu.RUnlock()

	if len(toRemove) > 0 {
		dt.mu.Lock()
		for _, id := range toRemove {
			delete(dt.deployments, id)
		}
		dt.mu.Unlock()
	}
}

// Count returns the number of tracked deployments
func (dt *DeploymentTracker) Count() int {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return len(dt.deployments)
}

// Shutdown performs cleanup and stops background goroutines
func (dt *DeploymentTracker) Shutdown(ctx context.Context) error {
	// Note: cleanupLoop will stop when the process exits
	// For a more graceful shutdown, we would need a done channel
	return nil
}

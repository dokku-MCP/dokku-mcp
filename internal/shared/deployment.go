package shared

import (
	"context"
	"time"
)

// DeploymentService defines the shared deployment capability
// This interface should be implemented by deployment plugins and consumed by other plugins
type DeploymentService interface {
	Deploy(ctx context.Context, appName string, options DeployOptions) (*DeploymentResult, error)
	Rollback(ctx context.Context, appName string, version string) error
	GetHistory(ctx context.Context, appName string) ([]DeploymentSummary, error)
	GetStatus(ctx context.Context, deploymentID string) (*DeploymentResult, error)
	Cancel(ctx context.Context, deploymentID string) error
}

// DeployOptions contains deployment configuration
type DeployOptions struct {
	RepoURL    string
	GitRef     *GitRef
	Buildpack  *BuildpackName
	BuildImage *DockerImage
	RunImage   *DockerImage
	Force      bool
}

// DeploymentResult represents the outcome of a deployment
type DeploymentResult struct {
	ID          string
	AppName     string
	GitRef      string
	Status      DeploymentStatus
	CreatedAt   time.Time
	CompletedAt *time.Time
	ErrorMsg    string
}

// DeploymentSummary provides a lightweight view of deployment history
type DeploymentSummary struct {
	ID        string
	GitRef    string
	Status    DeploymentStatus
	CreatedAt time.Time
	Duration  time.Duration
}

// DeploymentStatus represents deployment state
type DeploymentStatus string

const (
	DeploymentStatusPending    DeploymentStatus = "pending"
	DeploymentStatusRunning    DeploymentStatus = "running"
	DeploymentStatusSucceeded  DeploymentStatus = "succeeded"
	DeploymentStatusFailed     DeploymentStatus = "failed"
	DeploymentStatusRolledBack DeploymentStatus = "rolled_back"
)

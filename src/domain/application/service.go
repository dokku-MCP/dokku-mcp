package application

import (
	"context"
	"fmt"
	"time"
)

// DeploymentService interface pour les opérations de déploiement
type DeploymentService interface {
	Deploy(ctx context.Context, appName string, options DeployOptions) (*Deployment, error)
	Rollback(ctx context.Context, appName string, version string) error
	GetHistory(ctx context.Context, appName string) ([]*Deployment, error)
}

// DeployOptions options pour le déploiement
type DeployOptions struct {
	GitRef     string
	BuildPack  string
	ForceClean bool
	NoCache    bool
}

// Deployment représente un déploiement d'application
type Deployment struct {
	ID          string
	AppName     string
	GitRef      string
	Status      DeploymentStatus
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time
	ErrorMsg    string
	BuildLogs   string
}

// DeploymentStatus état d'un déploiement
type DeploymentStatus string

const (
	DeploymentStatusPending    DeploymentStatus = "pending"
	DeploymentStatusRunning    DeploymentStatus = "running"
	DeploymentStatusSucceeded  DeploymentStatus = "succeeded"
	DeploymentStatusFailed     DeploymentStatus = "failed"
	DeploymentStatusRolledBack DeploymentStatus = "rolled_back"
)

// NewDeployment crée un nouveau déploiement
func NewDeployment(appName, gitRef string) (*Deployment, error) {
	if appName == "" {
		return nil, fmt.Errorf("le nom de l'application ne peut pas être vide")
	}
	if gitRef == "" {
		gitRef = "main"
	}

	return &Deployment{
		ID:        generateDeploymentID(),
		AppName:   appName,
		GitRef:    gitRef,
		Status:    DeploymentStatusPending,
		CreatedAt: time.Now(),
	}, nil
}

// Start démarre le déploiement
func (d *Deployment) Start() {
	d.Status = DeploymentStatusRunning
	now := time.Now()
	d.StartedAt = &now
}

// Complete marque le déploiement comme terminé avec succès
func (d *Deployment) Complete() {
	d.Status = DeploymentStatusSucceeded
	now := time.Now()
	d.CompletedAt = &now
}

// Fail marque le déploiement comme échoué
func (d *Deployment) Fail(errorMsg string) {
	d.Status = DeploymentStatusFailed
	d.ErrorMsg = errorMsg
	now := time.Now()
	d.CompletedAt = &now
}

// Rollback marque le déploiement comme annulé
func (d *Deployment) Rollback() {
	d.Status = DeploymentStatusRolledBack
	now := time.Now()
	d.CompletedAt = &now
}

// IsRunning vérifie si le déploiement est en cours
func (d *Deployment) IsRunning() bool {
	return d.Status == DeploymentStatusRunning
}

// IsCompleted vérifie si le déploiement est terminé
func (d *Deployment) IsCompleted() bool {
	return d.Status == DeploymentStatusSucceeded ||
		d.Status == DeploymentStatusFailed ||
		d.Status == DeploymentStatusRolledBack
}

// Duration retourne la durée du déploiement
func (d *Deployment) Duration() time.Duration {
	if d.StartedAt == nil {
		return 0
	}

	endTime := time.Now()
	if d.CompletedAt != nil {
		endTime = *d.CompletedAt
	}

	return endTime.Sub(*d.StartedAt)
}

// generateDeploymentID génère un ID unique pour le déploiement
func generateDeploymentID() string {
	// Utilise un timestamp pour simplifier
	return fmt.Sprintf("deploy_%d", time.Now().UnixNano())
}

package domain

import (
	"fmt"
	"time"
)

// Deployment représente un déploiement d'application
type Deployment struct {
	id          string
	appName     string
	gitRef      string
	status      DeploymentStatus
	createdAt   time.Time
	startedAt   *time.Time
	completedAt *time.Time
	errorMsg    string
	buildLogs   string
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
		id:        generateDeploymentID(),
		appName:   appName,
		gitRef:    gitRef,
		status:    DeploymentStatusPending,
		createdAt: time.Now(),
	}, nil
}

// NewDeploymentWithTimestamp crée un nouveau déploiement avec un timestamp personnalisé
func NewDeploymentWithTimestamp(appName, gitRef string, createdAt time.Time) (*Deployment, error) {
	if appName == "" {
		return nil, fmt.Errorf("le nom de l'application ne peut pas être vide")
	}
	if gitRef == "" {
		gitRef = "main"
	}

	return &Deployment{
		id:        generateDeploymentID(),
		appName:   appName,
		gitRef:    gitRef,
		status:    DeploymentStatusPending,
		createdAt: createdAt,
	}, nil
}

// ID retourne l'identifiant du déploiement
func (d *Deployment) ID() string {
	return d.id
}

// AppName retourne le nom de l'application
func (d *Deployment) AppName() string {
	return d.appName
}

// GitRef retourne la référence Git
func (d *Deployment) GitRef() string {
	return d.gitRef
}

// Status retourne le statut du déploiement
func (d *Deployment) Status() DeploymentStatus {
	return d.status
}

// CreatedAt retourne la date de création
func (d *Deployment) CreatedAt() time.Time {
	return d.createdAt
}

// StartedAt retourne la date de début
func (d *Deployment) StartedAt() *time.Time {
	return d.startedAt
}

// CompletedAt retourne la date de fin
func (d *Deployment) CompletedAt() *time.Time {
	return d.completedAt
}

// ErrorMsg retourne le message d'erreur
func (d *Deployment) ErrorMsg() string {
	return d.errorMsg
}

// BuildLogs retourne les logs de construction
func (d *Deployment) BuildLogs() string {
	return d.buildLogs
}

// Start démarre le déploiement
func (d *Deployment) Start() {
	d.status = DeploymentStatusRunning
	now := time.Now()
	d.startedAt = &now
}

// Complete marque le déploiement comme terminé avec succès
func (d *Deployment) Complete() {
	d.status = DeploymentStatusSucceeded
	now := time.Now()
	d.completedAt = &now
}

// Fail marque le déploiement comme échoué
func (d *Deployment) Fail(errorMsg string) {
	d.status = DeploymentStatusFailed
	d.errorMsg = errorMsg
	now := time.Now()
	d.completedAt = &now
}

// Rollback marque le déploiement comme annulé
func (d *Deployment) Rollback() {
	d.status = DeploymentStatusRolledBack
	now := time.Now()
	d.completedAt = &now
}

// AddBuildLogs ajoute des logs de construction
func (d *Deployment) AddBuildLogs(logs string) {
	d.buildLogs += logs
}

// IsRunning vérifie si le déploiement est en cours
func (d *Deployment) IsRunning() bool {
	return d.status == DeploymentStatusRunning
}

// IsCompleted vérifie si le déploiement est terminé
func (d *Deployment) IsCompleted() bool {
	return d.status == DeploymentStatusSucceeded ||
		d.status == DeploymentStatusFailed ||
		d.status == DeploymentStatusRolledBack
}

// IsSuccessful vérifie si le déploiement a réussi
func (d *Deployment) IsSuccessful() bool {
	return d.status == DeploymentStatusSucceeded
}

// IsFailed vérifie si le déploiement a échoué
func (d *Deployment) IsFailed() bool {
	return d.status == DeploymentStatusFailed
}

// Duration retourne la durée du déploiement
func (d *Deployment) Duration() time.Duration {
	if d.startedAt == nil {
		return 0
	}

	endTime := time.Now()
	if d.completedAt != nil {
		endTime = *d.completedAt
	}

	return endTime.Sub(*d.startedAt)
}

// generateDeploymentID génère un ID unique pour le déploiement
func generateDeploymentID() string {
	// Utilise un timestamp pour simplifier
	return fmt.Sprintf("deploy_%d", time.Now().UnixNano())
}

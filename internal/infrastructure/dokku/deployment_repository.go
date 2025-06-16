package dokku

import (
	"context"
	"log/slog"
	"sort"
	"sync"

	"github.com/alex-galey/dokku-mcp/internal/domain/dokku/deployment"
)

type deploymentRepository struct {
	deployments map[string]*deployment.Deployment // id -> deployment
	appIndex    map[string][]string               // appName -> []deploymentIDs
	mutex       sync.RWMutex
	logger      *slog.Logger
}

func NewDeploymentRepository(logger *slog.Logger) deployment.DeploymentRepository {
	return &deploymentRepository{
		deployments: make(map[string]*deployment.Deployment),
		appIndex:    make(map[string][]string),
		logger:      logger,
	}
}

func (r *deploymentRepository) Save(ctx context.Context, deploy *deployment.Deployment) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.deployments[deploy.ID()] = deploy

	// Update app index
	appName := deploy.AppName()
	if _, exists := r.appIndex[appName]; !exists {
		r.appIndex[appName] = make([]string, 0)
	}

	// Check if deployment already indexed
	found := false
	for _, id := range r.appIndex[appName] {
		if id == deploy.ID() {
			found = true
			break
		}
	}

	if !found {
		r.appIndex[appName] = append(r.appIndex[appName], deploy.ID())
	}

	r.logger.Debug("Deployment saved", "id", deploy.ID(), "app", appName)
	return nil
}

func (r *deploymentRepository) FindByID(ctx context.Context, id string) (*deployment.Deployment, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	deploy, exists := r.deployments[id]
	if !exists {
		return nil, deployment.ErrDeploymentNotFound
	}

	return deploy, nil
}

func (r *deploymentRepository) FindByAppName(ctx context.Context, appName string) ([]*deployment.Deployment, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	deploymentIDs, exists := r.appIndex[appName]
	if !exists {
		return []*deployment.Deployment{}, nil
	}

	deployments := make([]*deployment.Deployment, 0, len(deploymentIDs))
	for _, id := range deploymentIDs {
		if deploy, exists := r.deployments[id]; exists {
			deployments = append(deployments, deploy)
		}
	}

	// Sort by creation time (most recent first)
	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].CreatedAt().After(deployments[j].CreatedAt())
	})

	return deployments, nil
}

func (r *deploymentRepository) FindAll(ctx context.Context) ([]*deployment.Deployment, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	deployments := make([]*deployment.Deployment, 0, len(r.deployments))
	for _, deploy := range r.deployments {
		deployments = append(deployments, deploy)
	}

	// Sort by creation time (most recent first)
	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].CreatedAt().After(deployments[j].CreatedAt())
	})

	return deployments, nil
}

func (r *deploymentRepository) Delete(ctx context.Context, id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	deploy, exists := r.deployments[id]
	if !exists {
		return deployment.ErrDeploymentNotFound
	}

	delete(r.deployments, id)

	// Remove from app index
	appName := deploy.AppName()
	if deploymentIDs, exists := r.appIndex[appName]; exists {
		for i, deploymentID := range deploymentIDs {
			if deploymentID == id {
				r.appIndex[appName] = append(deploymentIDs[:i], deploymentIDs[i+1:]...)
				break
			}
		}

		if len(r.appIndex[appName]) == 0 {
			delete(r.appIndex, appName)
		}
	}

	r.logger.Debug("Deployment deleted", "id", id, "app", appName)
	return nil
}

func (r *deploymentRepository) Update(ctx context.Context, deploy *deployment.Deployment) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.deployments[deploy.ID()]; !exists {
		return deployment.ErrDeploymentNotFound
	}

	r.deployments[deploy.ID()] = deploy
	r.logger.Debug("Deployment updated", "id", deploy.ID(), "app", deploy.AppName())
	return nil
}

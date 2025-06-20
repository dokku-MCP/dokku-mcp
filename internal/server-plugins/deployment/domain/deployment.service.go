package domain

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
)

// DeploymentService interface pour les opérations de déploiement
type DeploymentService interface {
	Deploy(ctx context.Context, appName string, options DeployOptions) (*Deployment, error)
	Rollback(ctx context.Context, appName string, version string) error
	GetHistory(ctx context.Context, appName string) ([]*Deployment, error)
	GetByID(ctx context.Context, deploymentID string) (*Deployment, error)
	Cancel(ctx context.Context, deploymentID string) error
}

// DeploymentInfrastructure simplified interface for infrastructure operations
type DeploymentInfrastructure interface {
	CheckApplicationExists(ctx context.Context, appName string) (bool, error)
	CreateApplication(ctx context.Context, appName string) error
	SetBuildpack(ctx context.Context, appName string, buildpack string) error
	PerformGitDeploy(ctx context.Context, appName string, gitRef string) error
	ParseDeploymentHistory(ctx context.Context, appName string) ([]*Deployment, error)
}

// DeployOptions simplified options for deployment
type DeployOptions struct {
	GitRef    string
	BuildPack string
}

// ApplicationDeploymentService implémentation du service de déploiement
type ApplicationDeploymentService struct {
	deploymentRepo DeploymentRepository
	infrastructure DeploymentInfrastructure
	logger         *slog.Logger
}

// NewApplicationDeploymentService crée une nouvelle instance du service
func NewApplicationDeploymentService(
	deploymentRepo DeploymentRepository,
	infrastructure DeploymentInfrastructure,
	logger *slog.Logger,
) *ApplicationDeploymentService {
	return &ApplicationDeploymentService{
		deploymentRepo: deploymentRepo,
		infrastructure: infrastructure,
		logger:         logger,
	}
}

// Deploy lance un déploiement d'application
func (s *ApplicationDeploymentService) Deploy(ctx context.Context, appName string, options DeployOptions) (*Deployment, error) {
	s.logger.Info("Démarrage du déploiement d'application",
		"nom_app", appName,
		"git_ref", options.GitRef)

	deployment, err := NewDeployment(appName, options.GitRef)
	if err != nil {
		return nil, fmt.Errorf("échec de création du déploiement: %w", err)
	}

	deployment.Start()

	exists, err := s.infrastructure.CheckApplicationExists(ctx, appName)
	if err != nil {
		deployment.Fail(fmt.Sprintf("Échec de vérification de l'existence de l'application: %v", err))
		return deployment, fmt.Errorf("échec de vérification de l'existence de l'application: %w", err)
	}

	if !exists {
		if err := s.infrastructure.CreateApplication(ctx, appName); err != nil {
			deployment.Fail(fmt.Sprintf("Échec de création de l'application: %v", err))
			return deployment, fmt.Errorf("échec de création de l'application: %w", err)
		}
		s.logger.Info("Application créée avec succès", "nom_app", appName)
	}

	if options.BuildPack != "" {
		if err := s.infrastructure.SetBuildpack(ctx, appName, options.BuildPack); err != nil {
			s.logger.Warn("Échec de définition du buildpack", "erreur", err)
		}
	}

	if err := s.infrastructure.PerformGitDeploy(ctx, appName, options.GitRef); err != nil {
		deployment.Fail(fmt.Sprintf("Échec du déploiement depuis git: %v", err))
		return deployment, fmt.Errorf("échec du déploiement depuis git: %w", err)
	}

	s.logger.Info("Déploiement Git terminé avec succès",
		"nom_app", appName,
		"git_ref", options.GitRef,
		"deployment_id", deployment.ID())

	deployment.Complete()

	if err := s.deploymentRepo.Save(ctx, deployment); err != nil {
		s.logger.Warn("Échec de sauvegarde du déploiement", "erreur", err)
	}

	s.logger.Info("Déploiement terminé avec succès",
		"nom_app", appName,
		"deployment_id", deployment.ID(),
		"durée", deployment.Duration())

	return deployment, nil
}

// Rollback effectue un rollback vers une version précédente
func (s *ApplicationDeploymentService) Rollback(ctx context.Context, appName string, version string) error {
	s.logger.Info("Démarrage du rollback d'application",
		"nom_app", appName,
		"version", version)

	deployments, err := s.deploymentRepo.FindByAppName(ctx, appName)
	if err != nil {
		return err
	}

	var targetDeployment *Deployment
	for _, d := range deployments {
		if d.ID() == version {
			targetDeployment = d
			break
		}
	}

	if targetDeployment == nil {
		return ErrDeploymentNotFound
	}

	if !targetDeployment.IsCompleted() {
		return fmt.Errorf("cannot rollback to incomplete deployment: %s", version)
	}

	rollbackDeploy, err := NewDeployment(appName, targetDeployment.GitRef())
	if err != nil {
		return err
	}

	rollbackDeploy.Rollback()

	// Perform the actual rollback
	if err := s.infrastructure.PerformGitDeploy(ctx, appName, targetDeployment.GitRef()); err != nil {
		rollbackDeploy.Fail(fmt.Sprintf("Échec du rollback: %v", err))
		_ = s.deploymentRepo.Save(ctx, rollbackDeploy)
		return fmt.Errorf("échec du rollback: %w", err)
	}

	rollbackDeploy.Complete()

	s.logger.Info("Rollback terminé avec succès",
		"nom_app", appName,
		"version", version)

	return s.deploymentRepo.Save(ctx, rollbackDeploy)
}

// GetHistory récupère l'historique des déploiements
func (s *ApplicationDeploymentService) GetHistory(ctx context.Context, appName string) ([]*Deployment, error) {
	s.logger.Debug("Récupération de l'historique des déploiements", "nom_app", appName)

	exists, err := s.infrastructure.CheckApplicationExists(ctx, appName)
	if err != nil {
		return nil, fmt.Errorf("échec de vérification de l'existence de l'application: %w", err)
	}
	if !exists {
		return []*Deployment{}, nil
	}

	deployments, err := s.infrastructure.ParseDeploymentHistory(ctx, appName)
	if err != nil {
		s.logger.Warn("Échec de récupération de l'historique depuis l'infrastructure",
			"erreur", err, "nom_app", appName)

		deployments, err = s.deploymentRepo.FindByAppName(ctx, appName)
		if err != nil {
			return nil, fmt.Errorf("échec de récupération de l'historique: %w", err)
		}
	}

	// Sort deployments by timestamp (most recent first)
	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].CreatedAt().After(deployments[j].CreatedAt())
	})

	s.logger.Debug("Historique des déploiements récupéré avec succès",
		"nom_app", appName, "nombre", len(deployments))

	return deployments, nil
}

// GetByID récupère un déploiement par son ID
func (s *ApplicationDeploymentService) GetByID(ctx context.Context, deploymentID string) (*Deployment, error) {
	s.logger.Debug("Récupération du déploiement par ID", "deployment_id", deploymentID)

	deploy, err := s.deploymentRepo.FindByID(ctx, deploymentID)
	if err == nil {
		return deploy, nil
	}

	return nil, ErrDeploymentNotFound
}

// Cancel annule un déploiement en cours
func (s *ApplicationDeploymentService) Cancel(ctx context.Context, deploymentID string) error {
	s.logger.Info("Annulation du déploiement", "deployment_id", deploymentID)

	deploy, err := s.deploymentRepo.FindByID(ctx, deploymentID)
	if err != nil {
		return err
	}

	if !deploy.IsRunning() {
		return ErrDeploymentNotRunning
	}

	deploy.Fail("Déploiement annulé par l'utilisateur")

	s.logger.Info("Déploiement annulé avec succès", "deployment_id", deploymentID)

	return s.deploymentRepo.Save(ctx, deploy)
}

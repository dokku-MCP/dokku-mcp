package usecases

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alex-galey/dokku-mcp/internal/domain/dokku/application"
	deployment_services "github.com/alex-galey/dokku-mcp/internal/domain/dokku/deployment/services"
)

// ApplicationUseCase orchestre les opérations sur les applications
type ApplicationUseCase struct {
	applicationRepo   application.ApplicationRepository
	deploymentSvc     deployment_services.DeploymentService
	validationService *application.ValidationService
	logger            *slog.Logger
}

// NewApplicationUseCase crée un nouveau use case d'application
func NewApplicationUseCase(
	applicationRepo application.ApplicationRepository,
	deploymentSvc deployment_services.DeploymentService,
	logger *slog.Logger,
) *ApplicationUseCase {
	return &ApplicationUseCase{
		applicationRepo:   applicationRepo,
		deploymentSvc:     deploymentSvc,
		validationService: application.NewValidationService(),
		logger:            logger,
	}
}

// CreateApplicationCommand représente les données pour créer une application
type CreateApplicationCommand struct {
	Name string
}

// CreateApplication orchestre la création d'une application
func (uc *ApplicationUseCase) CreateApplication(ctx context.Context, cmd CreateApplicationCommand) error {
	uc.logger.Info("Création d'application", "nom_app", cmd.Name)

	// Utiliser le service de validation du domaine
	validationResult := uc.validationService.ValidateApplicationName(ctx, cmd.Name)
	if !validationResult.IsValid {
		var errorMessages []string
		for _, validationError := range validationResult.Errors {
			errorMessages = append(errorMessages, validationError.Message)
		}
		return fmt.Errorf("validation échouée: %v", errorMessages)
	}

	// Log warnings if any
	if len(validationResult.Warnings) > 0 {
		for _, warning := range validationResult.Warnings {
			uc.logger.Warn("Avertissement de création",
				"field", warning.Field,
				"message", warning.Message,
				"code", warning.Code)
		}
	}

	// Créer l'entité application
	app, err := application.NewApplication(cmd.Name)
	if err != nil {
		return fmt.Errorf("impossible de créer l'application: %w", err)
	}

	// Vérifier si l'application existe déjà
	exists, err := uc.applicationRepo.Exists(ctx, app.Name())
	if err != nil {
		return fmt.Errorf("échec de vérification d'existence: %w", err)
	}
	if exists {
		return application.ErrApplicationAlreadyExists
	}

	// Sauvegarder l'application
	if err := uc.applicationRepo.Save(ctx, app); err != nil {
		return fmt.Errorf("échec de sauvegarde: %w", err)
	}

	uc.logger.Info("Application créée avec succès", "nom_app", cmd.Name)
	return nil
}

// DeployApplicationCommand représente les données pour déployer une application
type DeployApplicationCommand struct {
	Name   string
	GitRef string
}

// DeployApplication orchestre le déploiement d'une application
func (uc *ApplicationUseCase) DeployApplication(ctx context.Context, cmd DeployApplicationCommand) error {
	uc.logger.Info("Déploiement d'application",
		"nom_app", cmd.Name,
		"git_ref", cmd.GitRef)

	// Récupérer l'application
	appName, err := application.NewApplicationName(cmd.Name)
	if err != nil {
		return fmt.Errorf("nom d'application invalide: %w", err)
	}

	app, err := uc.applicationRepo.GetByName(ctx, appName)
	if err != nil {
		return fmt.Errorf("application introuvable: %w", err)
	}

	// Créer la référence Git pour la validation
	var gitRef *application.GitRef
	if cmd.GitRef != "" {
		gitRef, err = application.NewGitRef(cmd.GitRef)
		if err != nil {
			return fmt.Errorf("référence Git invalide: %w", err)
		}
	}

	// Utiliser le service de validation du domaine pour le déploiement
	validationResult := uc.validationService.ValidateDeployment(ctx, app, gitRef, "")
	if !validationResult.IsValid {
		var errorMessages []string
		for _, validationError := range validationResult.Errors {
			errorMessages = append(errorMessages, validationError.Message)
		}
		return fmt.Errorf("validation de déploiement échouée: %v", errorMessages)
	}

	// Log warnings if any
	if len(validationResult.Warnings) > 0 {
		for _, warning := range validationResult.Warnings {
			uc.logger.Warn("Avertissement de déploiement",
				"field", warning.Field,
				"message", warning.Message,
				"code", warning.Code)
		}
	}

	// Créer les options de déploiement
	deployOptions := deployment_services.DeployOptions{
		GitRef: cmd.GitRef,
	}

	// Effectuer le déploiement via le service domaine
	deployment, err := uc.deploymentSvc.Deploy(ctx, cmd.Name, deployOptions)
	if err != nil {
		return fmt.Errorf("échec du déploiement: %w", err)
	}

	// Sauvegarder les changements
	if err := uc.applicationRepo.Save(ctx, app); err != nil {
		uc.logger.Warn("Échec de sauvegarde après déploiement",
			"erreur", err)
	}

	uc.logger.Info("Déploiement terminé avec succès",
		"nom_app", cmd.Name,
		"deployment_id", deployment.ID())
	return nil
}

// ScaleApplicationCommand représente les données pour scaler une application
type ScaleApplicationCommand struct {
	Name        string
	ProcessType string
	Scale       int
}

// ScaleApplication orchestre le scaling d'une application
func (uc *ApplicationUseCase) ScaleApplication(ctx context.Context, cmd ScaleApplicationCommand) error {
	uc.logger.Info("Scaling d'application",
		"nom_app", cmd.Name,
		"process_type", cmd.ProcessType,
		"scale", cmd.Scale)

	// Récupérer l'application
	appName, err := application.NewApplicationName(cmd.Name)
	if err != nil {
		return fmt.Errorf("nom d'application invalide: %w", err)
	}

	app, err := uc.applicationRepo.GetByName(ctx, appName)
	if err != nil {
		return fmt.Errorf("application introuvable: %w", err)
	}

	// Créer le type de processus
	processType := application.ProcessType(cmd.ProcessType)

	// Utiliser le service de validation du domaine pour le scaling
	validationResult := uc.validationService.ValidateScale(ctx, app, processType, cmd.Scale)
	if !validationResult.IsValid {
		var errorMessages []string
		for _, validationError := range validationResult.Errors {
			errorMessages = append(errorMessages, validationError.Message)
		}
		return fmt.Errorf("validation de scaling échouée: %v", errorMessages)
	}

	// Log warnings if any
	if len(validationResult.Warnings) > 0 {
		for _, warning := range validationResult.Warnings {
			uc.logger.Warn("Avertissement de scaling",
				"field", warning.Field,
				"message", warning.Message,
				"code", warning.Code)
		}
	}

	// Effectuer le scaling en utilisant la méthode Scale de l'entité
	if err := app.Scale(processType, cmd.Scale); err != nil {
		return fmt.Errorf("échec du scaling: %w", err)
	}

	// Sauvegarder les changements
	if err := uc.applicationRepo.Save(ctx, app); err != nil {
		return fmt.Errorf("échec de sauvegarde après scaling: %w", err)
	}

	uc.logger.Info("Scaling terminé avec succès",
		"nom_app", cmd.Name,
		"process_type", cmd.ProcessType,
		"nouvelle_scale", cmd.Scale)
	return nil
}

// SetConfigCommand représente les données pour configurer une application
type SetConfigCommand struct {
	Name   string
	Config map[string]string
}

// SetApplicationConfig orchestre la configuration d'une application
func (uc *ApplicationUseCase) SetApplicationConfig(ctx context.Context, cmd SetConfigCommand) error {
	uc.logger.Info("Configuration d'application",
		"nom_app", cmd.Name,
		"nb_vars", len(cmd.Config))

	// Récupérer l'application
	appName, err := application.NewApplicationName(cmd.Name)
	if err != nil {
		return fmt.Errorf("nom d'application invalide: %w", err)
	}

	app, err := uc.applicationRepo.GetByName(ctx, appName)
	if err != nil {
		return fmt.Errorf("application introuvable: %w", err)
	}

	// Appliquer la configuration
	for key, value := range cmd.Config {
		if err := app.SetEnvironmentVariable(key, value); err != nil {
			return fmt.Errorf("impossible de définir la variable %s: %w", key, err)
		}
	}

	// Sauvegarder les changements
	if err := uc.applicationRepo.Save(ctx, app); err != nil {
		return fmt.Errorf("échec de sauvegarde après configuration: %w", err)
	}

	uc.logger.Info("Configuration appliquée avec succès",
		"nom_app", cmd.Name)
	return nil
}

// GetAllApplications récupère toutes les applications
func (uc *ApplicationUseCase) GetAllApplications(ctx context.Context) ([]*application.Application, error) {
	uc.logger.Debug("Récupération de toutes les applications")

	apps, err := uc.applicationRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("échec de récupération des applications: %w", err)
	}

	uc.logger.Debug("Applications récupérées avec succès",
		"nombre", len(apps))
	return apps, nil
}

// GetApplicationByName récupère une application par son nom
func (uc *ApplicationUseCase) GetApplicationByName(ctx context.Context, name string) (*application.Application, error) {
	uc.logger.Debug("Récupération d'application par nom",
		"nom_app", name)

	appName, err := application.NewApplicationName(name)
	if err != nil {
		return nil, fmt.Errorf("nom d'application invalide: %w", err)
	}

	app, err := uc.applicationRepo.GetByName(ctx, appName)
	if err != nil {
		return nil, fmt.Errorf("application introuvable: %w", err)
	}

	uc.logger.Debug("Application récupérée avec succès",
		"nom_app", name)
	return app, nil
}

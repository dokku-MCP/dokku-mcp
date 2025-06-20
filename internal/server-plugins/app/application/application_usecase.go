package usecases

import (
	"context"
	"fmt"
	"log/slog"

	domain "github.com/alex-galey/dokku-mcp/internal/server-plugins/app/domain"
	"github.com/alex-galey/dokku-mcp/internal/shared"
)

// ApplicationUseCase orchestrates application operations
type ApplicationUseCase struct {
	applicationRepo   domain.ApplicationRepository
	deploymentSvc     shared.DeploymentService
	validationService *domain.ValidationService
	logger            *slog.Logger
}

// NewApplicationUseCase creates a new application use case
func NewApplicationUseCase(
	applicationRepo domain.ApplicationRepository,
	deploymentSvc shared.DeploymentService,
	logger *slog.Logger,
) *ApplicationUseCase {
	return &ApplicationUseCase{
		applicationRepo:   applicationRepo,
		deploymentSvc:     deploymentSvc,
		validationService: domain.NewValidationService(),
		logger:            logger,
	}
}

// CreateApplicationCommand represents the data for creating an application
type CreateApplicationCommand struct {
	Name string
}

// CreateApplication orchestrates application creation
func (uc *ApplicationUseCase) CreateApplication(ctx context.Context, cmd CreateApplicationCommand) error {
	uc.logger.Info("Creating application", "app_name", cmd.Name)

	// Use domain validation service
	validationResult := uc.validationService.ValidateApplicationName(ctx, cmd.Name)
	if !validationResult.IsValid {
		var errorMessages []string
		for _, validationError := range validationResult.Errors {
			errorMessages = append(errorMessages, validationError.Message)
		}
		return fmt.Errorf("validation failed: %v", errorMessages)
	}

	// Log warnings if any
	if len(validationResult.Warnings) > 0 {
		for _, warning := range validationResult.Warnings {
			uc.logger.Warn("Creation warning",
				"field", warning.Field,
				"message", warning.Message,
				"code", warning.Code)
		}
	}

	// Create application entity
	app, err := domain.NewApplication(cmd.Name)
	if err != nil {
		return fmt.Errorf("unable to create application: %w", err)
	}

	// Check if application already exists
	exists, err := uc.applicationRepo.Exists(ctx, app.Name())
	if err != nil {
		return fmt.Errorf("failed to check existence: %w", err)
	}
	if exists {
		return domain.ErrApplicationAlreadyExists
	}

	// Save application
	if err := uc.applicationRepo.Save(ctx, app); err != nil {
		return fmt.Errorf("failed to save: %w", err)
	}

	uc.logger.Info("Application created successfully", "app_name", cmd.Name)
	return nil
}

// DeployApplicationCommand represents the data for deploying an application
type DeployApplicationCommand struct {
	Name   string
	GitRef string
}

// DeployApplication orchestrates application deployment
func (uc *ApplicationUseCase) DeployApplication(ctx context.Context, cmd DeployApplicationCommand) error {
	uc.logger.Info("Deploying application",
		"app_name", cmd.Name,
		"git_ref", cmd.GitRef)

	// Get application
	appName, err := domain.NewApplicationName(cmd.Name)
	if err != nil {
		return fmt.Errorf("invalid application name: %w", err)
	}

	app, err := uc.applicationRepo.GetByName(ctx, appName)
	if err != nil {
		return fmt.Errorf("application not found: %w", err)
	}

	// Create Git reference for validation
	var gitRef *shared.GitRef
	if cmd.GitRef != "" {
		gitRef, err = shared.NewGitRef(cmd.GitRef)
		if err != nil {
			return fmt.Errorf("invalid Git reference: %w", err)
		}
	}

	// Use domain validation service for deployment
	validationResult := uc.validationService.ValidateDeployment(ctx, app, gitRef, "")
	if !validationResult.IsValid {
		var errorMessages []string
		for _, validationError := range validationResult.Errors {
			errorMessages = append(errorMessages, validationError.Message)
		}
		return fmt.Errorf("deployment validation failed: %v", errorMessages)
	}

	// Log warnings if any
	if len(validationResult.Warnings) > 0 {
		for _, warning := range validationResult.Warnings {
			uc.logger.Warn("Deployment warning",
				"field", warning.Field,
				"message", warning.Message,
				"code", warning.Code)
		}
	}

	// Create deployment options using shared interface
	deployOptions := shared.DeployOptions{
		GitRef: cmd.GitRef,
	}

	// Perform deployment via shared service interface
	deploymentResult, err := uc.deploymentSvc.Deploy(ctx, cmd.Name, deployOptions)
	if err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}

	// Save changes
	if err := uc.applicationRepo.Save(ctx, app); err != nil {
		uc.logger.Warn("Failed to save after deployment",
			"error", err)
	}

	uc.logger.Info("Deployment completed successfully",
		"app_name", cmd.Name,
		"deployment_id", deploymentResult.ID)
	return nil
}

// ScaleApplicationCommand represents the data for scaling an application
type ScaleApplicationCommand struct {
	Name        string
	ProcessType string
	Scale       int
}

// ScaleApplication orchestrates application scaling
func (uc *ApplicationUseCase) ScaleApplication(ctx context.Context, cmd ScaleApplicationCommand) error {
	uc.logger.Info("Scaling application",
		"app_name", cmd.Name,
		"process_type", cmd.ProcessType,
		"scale", cmd.Scale)

	// Get application
	appName, err := domain.NewApplicationName(cmd.Name)
	if err != nil {
		return fmt.Errorf("invalid application name: %w", err)
	}

	app, err := uc.applicationRepo.GetByName(ctx, appName)
	if err != nil {
		return fmt.Errorf("application not found: %w", err)
	}

	// Create process type
	processType := domain.ProcessType(cmd.ProcessType)

	// Use domain validation service for scaling
	validationResult := uc.validationService.ValidateScale(ctx, app, processType, cmd.Scale)
	if !validationResult.IsValid {
		var errorMessages []string
		for _, validationError := range validationResult.Errors {
			errorMessages = append(errorMessages, validationError.Message)
		}
		return fmt.Errorf("scaling validation failed: %v", errorMessages)
	}

	// Log warnings if any
	if len(validationResult.Warnings) > 0 {
		for _, warning := range validationResult.Warnings {
			uc.logger.Warn("Scaling warning",
				"field", warning.Field,
				"message", warning.Message,
				"code", warning.Code)
		}
	}

	// Scale application via domain entity
	if err := app.Scale(processType, cmd.Scale); err != nil {
		return fmt.Errorf("scaling failed: %w", err)
	}

	// Save changes
	if err := uc.applicationRepo.Save(ctx, app); err != nil {
		uc.logger.Warn("Failed to save after scaling",
			"error", err)
	}

	uc.logger.Info("Scaling completed successfully",
		"app_name", cmd.Name,
		"process_type", cmd.ProcessType,
		"scale", cmd.Scale)
	return nil
}

// SetConfigCommand represents the data for configuring an application
type SetConfigCommand struct {
	Name   string
	Config map[string]string
}

// SetApplicationConfig orchestrates application configuration
func (uc *ApplicationUseCase) SetApplicationConfig(ctx context.Context, cmd SetConfigCommand) error {
	uc.logger.Info("Configuring application",
		"app_name", cmd.Name,
		"nb_vars", len(cmd.Config))

	// Get application
	appName, err := domain.NewApplicationName(cmd.Name)
	if err != nil {
		return fmt.Errorf("invalid application name: %w", err)
	}

	app, err := uc.applicationRepo.GetByName(ctx, appName)
	if err != nil {
		return fmt.Errorf("application not found: %w", err)
	}

	// Apply configuration
	for key, value := range cmd.Config {
		if err := app.SetEnvironmentVariable(key, value); err != nil {
			return fmt.Errorf("unable to set variable %s: %w", key, err)
		}
	}

	// Save changes
	if err := uc.applicationRepo.Save(ctx, app); err != nil {
		return fmt.Errorf("failed to save after configuration: %w", err)
	}

	uc.logger.Info("Configuration applied successfully",
		"app_name", cmd.Name)
	return nil
}

// GetAllApplications retrieves all applications
func (uc *ApplicationUseCase) GetAllApplications(ctx context.Context) ([]*domain.Application, error) {
	uc.logger.Debug("Retrieving all applications")

	apps, err := uc.applicationRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve applications: %w", err)
	}

	uc.logger.Debug("Applications retrieved successfully",
		"count", len(apps))
	return apps, nil
}

// GetApplicationByName retrieves an application by its name
func (uc *ApplicationUseCase) GetApplicationByName(ctx context.Context, name string) (*domain.Application, error) {
	uc.logger.Debug("Retrieving application by name",
		"app_name", name)

	appName, err := domain.NewApplicationName(name)
	if err != nil {
		return nil, fmt.Errorf("invalid application name: %w", err)
	}

	app, err := uc.applicationRepo.GetByName(ctx, appName)
	if err != nil {
		return nil, fmt.Errorf("application not found: %w", err)
	}

	uc.logger.Debug("Application retrieved successfully",
		"app_name", name)
	return app, nil
}

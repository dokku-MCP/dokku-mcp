package usecases

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	domain "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/app/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/shared"
	"github.com/dokku-mcp/dokku-mcp/internal/shared/process"
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
	Name       string
	RepoURL    string
	GitRef     string
	BuildImage string
	RunImage   string
}

// DeployApplication orchestrates application deployment. The build runs
// asynchronously; the returned result identifies the tracked deployment.
func (uc *ApplicationUseCase) DeployApplication(ctx context.Context, cmd DeployApplicationCommand) (*shared.DeploymentResult, error) {
	uc.logger.Info("Deploying application",
		"app_name", cmd.Name,
		"repo_url", cmd.RepoURL,
		"git_ref", cmd.GitRef)

	// Get application
	appName, err := domain.NewApplicationName(cmd.Name)
	if err != nil {
		return nil, fmt.Errorf("invalid application name: %w", err)
	}

	app, err := uc.applicationRepo.GetByName(ctx, appName)
	if err != nil {
		return nil, fmt.Errorf("application not found: %w", err)
	}

	// Create Git reference for validation
	var gitRef *shared.GitRef
	if cmd.GitRef != "" {
		var err error
		gitRef, err = shared.NewGitRef(cmd.GitRef)
		if err != nil {
			return nil, fmt.Errorf("invalid Git reference: %w", err)
		}
	}

	// Use domain validation service for deployment
	validationResult := uc.validationService.ValidateDeployment(ctx, app, gitRef, "")
	if !validationResult.IsValid {
		var errorMessages []string
		for _, validationError := range validationResult.Errors {
			errorMessages = append(errorMessages, validationError.Message)
		}
		return nil, fmt.Errorf("deployment validation failed: %v", errorMessages)
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

	var buildImage, runImage *shared.DockerImage
	if cmd.BuildImage != "" {
		buildImage, err = shared.NewDockerImage(cmd.BuildImage)
		if err != nil {
			return nil, fmt.Errorf("invalid build image: %w", err)
		}
	}
	if cmd.RunImage != "" {
		runImage, err = shared.NewDockerImage(cmd.RunImage)
		if err != nil {
			return nil, fmt.Errorf("invalid run image: %w", err)
		}
	}

	// Create deployment options using shared interface
	deployOptions := shared.DeployOptions{
		RepoURL:    cmd.RepoURL,
		GitRef:     gitRef,
		BuildImage: buildImage,
		RunImage:   runImage,
	}

	// Perform deployment via shared service interface
	deploymentResult, err := uc.deploymentSvc.Deploy(ctx, cmd.Name, deployOptions)
	if err != nil {
		uc.logger.Error("Deployment service failed", "app_name", cmd.Name, "error", err)
		// Rollback app state
		if failErr := app.FailDeployment(err.Error()); failErr != nil {
			uc.logger.Error("failed to mark deployment as failed", "error", failErr)
		}
		if saveErr := uc.applicationRepo.Save(ctx, app); saveErr != nil {
			uc.logger.Error("failed to save app state after deployment failure", "error", saveErr)
		}
		return nil, fmt.Errorf("deployment failed: %w", err)
	}

	// Update domain entity
	if err := app.Deploy(gitRef, &domain.DeploymentOptions{
		BuildImage: buildImage,
		RunImage:   runImage,
	}); err != nil {
		return nil, fmt.Errorf("failed to update application state: %w", err)
	}

	// Save changes
	if err := uc.applicationRepo.Save(ctx, app); err != nil {
		uc.logger.Warn("Failed to save after deployment",
			"error", err)
	}

	uc.logger.Info("Deployment started",
		"app_name", cmd.Name,
		"deployment_id", deploymentResult.ID)
	return deploymentResult, nil
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
	processType, err := process.NewProcessType(cmd.ProcessType)
	if err != nil {
		return fmt.Errorf("invalid process type: %w", err)
	}

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
	// Restart restarts the application so that it picks up the new values.
	Restart bool
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

	if err := app.Configure(cmd.Config, cmd.Restart); err != nil {
		return err
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

// GetApplicationLogs returns the last lines of an application's runtime logs.
func (uc *ApplicationUseCase) GetApplicationLogs(ctx context.Context, name string, lines int) (string, error) {
	appName, err := uc.existingApp(ctx, name)
	if err != nil {
		return "", err
	}
	return uc.applicationRepo.GetLogs(ctx, appName, lines)
}

// ChangeProcessState restarts, stops or starts all processes of an application.
func (uc *ApplicationUseCase) ChangeProcessState(ctx context.Context, name string, action domain.ProcessAction) error {
	appName, err := uc.existingApp(ctx, name)
	if err != nil {
		return err
	}
	uc.logger.Info("Changing application process state", "app_name", name, "action", action)
	return uc.applicationRepo.SetProcessState(ctx, appName, action)
}

// GetFailedDeployLogs returns the logs of the containers of the last failed deploy.
func (uc *ApplicationUseCase) GetFailedDeployLogs(ctx context.Context, name string) (string, error) {
	appName, err := uc.existingApp(ctx, name)
	if err != nil {
		return "", err
	}
	return uc.applicationRepo.GetFailedDeployLogs(ctx, appName)
}

// RollbackApplicationCommand redeploys an earlier Git reference.
type RollbackApplicationCommand struct {
	Name   string
	GitRef string
	// RepoURL defaults to the repository of the last git:sync deployment.
	RepoURL string
}

// ErrUnknownDeploySource is returned when a rollback cannot infer the
// repository the application was deployed from.
var ErrUnknownDeploySource = errors.New("cannot determine the repository the application was deployed from; pass repo_url")

// RollbackApplication redeploys an earlier Git reference. Dokku keeps no
// release history, so a rollback is a deployment of a known-good ref from
// the same repository.
func (uc *ApplicationUseCase) RollbackApplication(ctx context.Context, cmd RollbackApplicationCommand) (*shared.DeploymentResult, error) {
	repoURL := cmd.RepoURL
	if repoURL == "" {
		appName, err := uc.existingApp(ctx, cmd.Name)
		if err != nil {
			return nil, err
		}
		source, err := uc.applicationRepo.GetDeploySource(ctx, appName)
		if err != nil {
			return nil, err
		}
		repoURL = source.RepoURL()
		if repoURL == "" {
			return nil, ErrUnknownDeploySource
		}
	}
	uc.logger.Info("Rolling back application", "app_name", cmd.Name, "git_ref", cmd.GitRef, "repo_url", repoURL)
	return uc.DeployApplication(ctx, DeployApplicationCommand{Name: cmd.Name, RepoURL: repoURL, GitRef: cmd.GitRef})
}

// existingApp validates a name and checks that the application exists.
func (uc *ApplicationUseCase) existingApp(ctx context.Context, name string) (*domain.ApplicationName, error) {
	appName, err := domain.NewApplicationName(name)
	if err != nil {
		return nil, fmt.Errorf("invalid application name: %w", err)
	}
	exists, err := uc.applicationRepo.Exists(ctx, appName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrApplicationNotFound
	}
	return appName, nil
}

// FollowLogsResult holds the lines collected while following logs.
type FollowLogsResult struct {
	Lines     []string
	Truncated bool
	Elapsed   time.Duration
}

// FollowLogs collects new log lines for up to duration or maxLines lines,
// calling onLine for each line as it arrives.
func (uc *ApplicationUseCase) FollowLogs(ctx context.Context, name string, duration time.Duration, maxLines int, onLine func(string)) (*FollowLogsResult, error) {
	appName, err := uc.existingApp(ctx, name)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	lines, errs, err := uc.applicationRepo.FollowLogs(ctx, appName)
	if err != nil {
		return nil, err
	}
	result := &FollowLogsResult{Lines: []string{}}
	for line := range lines {
		result.Lines = append(result.Lines, line)
		if onLine != nil {
			onLine(line)
		}
		if len(result.Lines) >= maxLines {
			result.Truncated = true
			cancel()
			break
		}
	}
	result.Elapsed = time.Since(start)
	if !result.Truncated {
		if err := <-errs; err != nil {
			return nil, err
		}
	}
	return result, nil
}

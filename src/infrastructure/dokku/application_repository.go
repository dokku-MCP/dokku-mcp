package dokku

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/alex-galey/dokku-mcp/src/domain/application"
)

// applicationRepository implements application.Repository with Dokku
type applicationRepository struct {
	client DokkuClient
	logger *slog.Logger
}

func NewApplicationRepository(client DokkuClient, logger *slog.Logger) application.Repository {
	return &applicationRepository{
		client: client,
		logger: logger,
	}
}

func (r *applicationRepository) GetAll(ctx context.Context) ([]*application.Application, error) {
	r.logger.Debug("Retrieving all applications")

	appNames, err := r.client.GetApplications(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get application names: %w", err)
	}

	applications := make([]*application.Application, 0, len(appNames))

	for _, appName := range appNames {
		app, err := r.GetByName(ctx, appName)
		if err != nil {
			r.logger.Warn("Failed to retrieve application",
				"error", err,
				"app_name", appName)
			continue
		}
		applications = append(applications, app)
	}

	r.logger.Debug("Applications retrieved successfully",
		"number", len(applications))
	return applications, nil
}

func (r *applicationRepository) GetByName(ctx context.Context, name string) (*application.Application, error) {
	r.logger.Debug("Retrieving application by name",
		"app_name", name)

	app, err := application.NewApplication(name)
	if err != nil {
		return nil, fmt.Errorf("failed to create application entity: %w", err)
	}

	exists, err := r.Exists(ctx, name)
	if err != nil {
		r.logger.Warn("Unable to verify application existence",
			"error", err,
			"app_name", name)
	} else if !exists {
		r.logger.Warn("Application does not exist in Dokku despite being present in list",
			"app_name", name)
		// Return basic application even if it doesn't really exist
		r.logger.Debug("Application retrieved with minimal information (phantom app)",
			"app_name", name)
		return app, nil
	}

	info, err := r.client.GetApplicationInfo(ctx, name)
	if err != nil {
		r.logger.Warn("Failed to retrieve detailed application information - using basic information",
			"error", err,
			"app_name", name,
			"probable_reason", "application not deployed or information unavailable")

		// Try to get basic information via apps:report if available
		if reportInfo, reportErr := r.tryGetBasicApplicationInfo(ctx, name); reportErr == nil {
			r.logger.Debug("Basic information retrieved via apps:report",
				"app_name", name)
			if err := r.updateApplicationFromInfo(app, reportInfo, make(map[string]string)); err != nil {
				r.logger.Warn("Failed to update from apps:report",
					"error", err,
					"app_name", name)
			}
		}

		r.logger.Debug("Application retrieved with basic information",
			"app_name", name)
		return app, nil
	}

	config, err := r.client.GetApplicationConfig(ctx, name)
	if err != nil {
		r.logger.Warn("Failed to retrieve application configuration - using empty configuration",
			"error", err,
			"app_name", name)
		config = make(map[string]string)
	}

	if err := r.updateApplicationFromInfo(app, info, config); err != nil {
		r.logger.Warn("Failed to update application from Dokku information",
			"error", err,
			"app_name", name)
	}

	r.logger.Debug("Application retrieved successfully",
		"app_name", name)
	return app, nil
}

func (r *applicationRepository) Save(ctx context.Context, app *application.Application) error {
	r.logger.Debug("Saving application",
		"app_name", app.Name())

	exists, err := r.Exists(ctx, app.Name())
	if err != nil {
		return fmt.Errorf("failed to check application existence: %w", err)
	}

	if !exists {
		_, err := r.client.ExecuteCommand(ctx, "apps:create", []string{app.Name()})
		if err != nil {
			return fmt.Errorf("failed to create application: %w", err)
		}
	}

	// Update the configuration
	configMap := make(map[string]string)
	for key, value := range app.Config().EnvironmentVars {
		configMap[key] = value
	}

	if len(configMap) > 0 {
		if err := r.client.SetApplicationConfig(ctx, app.Name(), configMap); err != nil {
			return fmt.Errorf("failed to update configuration: %w", err)
		}
	}

	r.logger.Debug("Application saved successfully",
		"app_name", app.Name())
	return nil
}

func (r *applicationRepository) Delete(ctx context.Context, name string) error {
	r.logger.Debug("Deleting application",
		"app_name", name)

	_, err := r.client.ExecuteCommand(ctx, "apps:destroy", []string{name, "--force"})
	if err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}

	r.logger.Debug("Application deleted successfully",
		"app_name", name)
	return nil
}

func (r *applicationRepository) Exists(ctx context.Context, name string) (bool, error) {
	r.logger.Debug("Checking application existence",
		"app_name", name)

	_, err := r.client.ExecuteCommand(ctx, "apps:exists", []string{name})
	if err != nil {
		return false, nil
	}

	return true, nil
}

func (r *applicationRepository) List(ctx context.Context, offset, limit int) ([]*application.Application, int, error) {
	r.logger.Debug("Retrieving paginated list of applications",
		"offset", offset,
		"limit", limit)

	allApps, err := r.GetAll(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get all applications: %w", err)
	}

	total := len(allApps)

	start := min(offset, total)
	end := min(start+limit, total)

	pagedApps := allApps[start:end]

	r.logger.Debug("Paginated list retrieved successfully",
		"total", total,
		"returned", len(pagedApps))

	return pagedApps, total, nil
}

func (r *applicationRepository) GetByState(ctx context.Context, state application.ApplicationState) ([]*application.Application, error) {
	r.logger.Debug("Retrieving applications by state",
		"state", state)

	allApps, err := r.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all applications: %w", err)
	}

	var filteredApps []*application.Application
	for _, app := range allApps {
		if app.State() == state {
			filteredApps = append(filteredApps, app)
		}
	}

	r.logger.Debug("Applications filtered by state retrieved successfully",
		"state", state,
		"number", len(filteredApps))

	return filteredApps, nil
}

// Update an application with the Dokku information
func (r *applicationRepository) updateApplicationFromInfo(app *application.Application, info map[string]interface{}, config map[string]string) error {
	// Update the environment variables
	appConfig := app.Config()
	for key, value := range config {
		if err := appConfig.SetEnvironmentVar(key, value); err != nil {
			r.logger.Warn("Failed to set environment variable",
				"error", err,
				"key", key,
				"value", value)
		}
	}

	// Try to determine the state from the information
	if deployedStr, exists := info["Deployed"]; exists {
		if deployed, ok := deployedStr.(string); ok && deployed != "" {
			if err := app.UpdateState(application.StateDeployed); err != nil {
				r.logger.Warn("Failed to update state",
					"error", err)
			}
		}
	}

	// Process the processes information if it exists
	if processesStr, exists := info["Processes"]; exists {
		if processes, ok := processesStr.(string); ok {
			r.parseProcesses(app, processes)
		}
	}

	return nil
}

// Parses the processes information and updates the application
func (r *applicationRepository) parseProcesses(app *application.Application, processesStr string) {
	lines := strings.Split(processesStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Typical format: "web: 1"
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		processType := strings.TrimSpace(parts[0])
		scaleStr := strings.TrimSpace(parts[1])

		scale, err := strconv.Atoi(scaleStr)
		if err != nil {
			continue
		}

		// Map the process types
		var mappedType application.ProcessType
		switch processType {
		case "web":
			mappedType = application.ProcessTypeWeb
		case "worker":
			mappedType = application.ProcessTypeWorker
		case "cron":
			mappedType = application.ProcessTypeCron
		case "release":
			mappedType = application.ProcessTypeRelease
		default:
			// Use "web" by default for unknown types
			mappedType = application.ProcessTypeWeb
		}

		// Create and add the process
		process, err := application.NewProcess(mappedType, "", scale)
		if err != nil {
			r.logger.Warn("Failed to create process",
				"error", err,
				"type_process", processType)
			continue
		}

		if err := app.Config().AddProcess(process); err != nil {
			r.logger.Warn("Failed to add process",
				"error", err,
				"type_process", processType)
		}
	}
}

// via apps:report when apps:info fails
func (r *applicationRepository) tryGetBasicApplicationInfo(ctx context.Context, appName string) (map[string]interface{}, error) {
	// Check if apps:report is in allowed commands
	output, err := r.client.ExecuteCommand(ctx, "apps:report", []string{appName})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve application report: %w", err)
	}

	// Parse apps:report output to extract basic information
	info := make(map[string]any)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				// Normalize keys to match apps:info format
				switch {
				case strings.Contains(strings.ToLower(key), "deployed"):
					info["Deployed"] = value
				case strings.Contains(strings.ToLower(key), "git"):
					info["Git"] = value
				case strings.Contains(strings.ToLower(key), "running"):
					info["Running"] = value
				default:
					info[key] = value
				}
			}
		}
	}

	r.logger.Debug("Basic information extracted from apps:report",
		"app_name", appName,
		"fields_count", len(info))

	return info, nil
}

package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	dokkuApi "github.com/dokku-mcp/dokku-mcp/internal/dokku-api"
	app "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/app/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/shared/process"
)

// DokkuApplicationRepository implements the repository for applications via Dokku
type DokkuApplicationRepository struct {
	client dokkuApi.DokkuClient
	dokku  *DokkuApplicationAdapter
	logger *slog.Logger
}

// NewDokkuApplicationRepository creates a new application repository
func NewDokkuApplicationRepository(client dokkuApi.DokkuClient, logger *slog.Logger) app.ApplicationRepository {
	return &DokkuApplicationRepository{
		client: client,
		dokku:  NewDokkuApplicationAdapter(client, logger),
		logger: logger,
	}
}

// GetAll retrieves all applications
func (r *DokkuApplicationRepository) GetAll(ctx context.Context) ([]*app.Application, error) {
	r.logger.Debug("Retrieving all applications")

	appNames, err := r.dokku.GetApplications(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve application names: %w", err)
	}

	applications := make([]*app.Application, 0, len(appNames))

	for _, appName := range appNames {
		appNameVO, err := app.NewApplicationName(appName)
		if err != nil {
			r.logger.Warn("Invalid application name, skipped",
				"error", err,
				"app_name", appName)
			continue
		}

		appInstance, err := r.GetByName(ctx, appNameVO)
		if err != nil {
			r.logger.Warn("Failed to retrieve application",
				"error", err,
				"app_name", appName)
			continue
		}
		applications = append(applications, appInstance)
	}

	r.logger.Debug("Applications retrieved successfully",
		"count", len(applications))
	return applications, nil
}

// GetByName retrieves an application by its name
func (r *DokkuApplicationRepository) GetByName(ctx context.Context, name *app.ApplicationName) (*app.Application, error) {
	r.logger.Debug("Retrieving application by name",
		"app_name", name.Value())

	// Check if the application exists in Dokku first
	exists, err := r.Exists(ctx, name)
	if err != nil {
		return nil, err
	}
	if !exists {
		r.logger.Warn("Application does not exist in Dokku",
			"app_name", name.Value())
		return nil, app.ErrApplicationNotFound
	}

	// Get ps:report information for proper state detection
	info, err := r.tryGetPsReportInfo(ctx, name.Value())
	if err != nil {
		r.logger.Warn("Failed to retrieve ps:report - using basic info",
			"error", err,
			"app_name", name.Value())

		// Try to get basic information via apps:report as fallback
		if reportInfo, reportErr := r.tryGetBasicApplicationInfo(ctx, name.Value()); reportErr == nil {
			info = reportInfo
		} else {
			info = make(map[string]string)
		}
	}

	// Determine state from Dokku output
	state := r.determineStateFromInfo(info)

	// Create application entity with the determined state
	appInstance, err := app.NewApplicationWithState(name.Value(), state)
	if err != nil {
		return nil, fmt.Errorf("failed to create application entity: %w", err)
	}

	// Get configuration
	config, err := r.dokku.GetApplicationConfig(ctx, name.Value())
	if err != nil {
		r.logger.Warn("Failed to retrieve configuration - using empty configuration",
			"error", err,
			"app_name", name.Value())
		config = make(map[string]string)
	}

	// Update application with retrieved information
	if err := r.updateApplicationFromInfo(appInstance, info, config); err != nil {
		r.logger.Warn("Failed to update application from Dokku information",
			"error", err,
			"app_name", name.Value())
	}

	r.logger.Debug("Application retrieved successfully",
		"app_name", name.Value(),
		"state", state)
	return appInstance, nil
}

// Save saves an application
func (r *DokkuApplicationRepository) Save(ctx context.Context, application *app.Application) error {
	r.logger.Debug("Saving application",
		"app_name", application.Name().Value())

	exists, err := r.Exists(ctx, application.Name())
	if err != nil {
		return fmt.Errorf("failed to check application existence: %w", err)
	}

	if !exists {
		_, err := r.dokku.ExecuteCommand(ctx, app.CommandAppsCreate, []string{application.Name().Value()})
		if err != nil {
			return fmt.Errorf("failed to create application: %w", err)
		}
	}

	for _, event := range application.GetEvents() {
		switch e := event.(type) {
		case *app.ApplicationScaledEvent:
			if err := r.dokku.ScaleApplication(ctx, e.AggregateID(), e.ProcessType(), e.NewScale()); err != nil {
				r.logger.Error("Failed to apply scaling event", "error", err)
				return fmt.Errorf("failed to scale application during save: %w", err)
			}
			r.logger.Debug("Applied scaling event", "app", e.AggregateID(), "process", e.ProcessType(), "scale", e.NewScale())
		case *app.ConfigurationChangedEvent:
			if err := r.dokku.SetApplicationConfig(ctx, e.AggregateID(), e.Vars(), e.Restart()); err != nil {
				return fmt.Errorf("failed to update configuration: %w", err)
			}
			r.logger.Debug("Applied configuration event", "app", e.AggregateID(), "vars", len(e.Vars()))
		}
	}
	application.ClearEvents()

	r.logger.Debug("Application saved successfully",
		"app_name", application.Name().Value())
	return nil
}

// Delete deletes an application
func (r *DokkuApplicationRepository) Delete(ctx context.Context, name *app.ApplicationName) error {
	r.logger.Debug("Deleting application",
		"app_name", name.Value())

	_, err := r.dokku.ExecuteCommand(ctx, app.CommandAppsDestroy, []string{name.Value(), "--force"})
	if err != nil {
		return fmt.Errorf("failed to delete application: %w", err)
	}

	r.logger.Debug("Application deleted successfully",
		"app_name", name.Value())
	return nil
}

// Exists checks if an application exists
func (r *DokkuApplicationRepository) Exists(ctx context.Context, name *app.ApplicationName) (bool, error) {
	r.logger.Debug("Checking application existence",
		"app_name", name.Value())

	_, err := r.dokku.ExecuteCommand(ctx, app.CommandAppsExists, []string{name.Value()})
	if err == nil {
		return true, nil
	}
	// Only Dokku's own "App ... does not exist" answer means the app is
	// missing; transport, permission or other failures must not be read as
	// "missing" (Save would then try apps:create).
	if dokkuApi.IsNotFoundError(err) {
		return false, nil
	}
	return false, fmt.Errorf("cannot check whether application %s exists: %w", name.Value(), err)
}

// GetLogs returns the last lines of the application's runtime logs
func (r *DokkuApplicationRepository) GetLogs(ctx context.Context, name *app.ApplicationName, lines int) (string, error) {
	return r.dokku.GetApplicationLogs(ctx, name.Value(), lines)
}

// FollowLogs streams new runtime log lines until ctx is cancelled
func (r *DokkuApplicationRepository) FollowLogs(ctx context.Context, name *app.ApplicationName) (<-chan string, <-chan error, error) {
	logLines, errs, err := r.dokku.StreamLogs(ctx, name.Value())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to follow logs: %w", err)
	}
	out := make(chan string)
	outErr := make(chan error, 1)
	go func() {
		defer close(outErr)
		defer close(out)
		for line := range logLines {
			text := line.Message
			if line.Container != "" {
				text = line.Timestamp.Format(time.RFC3339) + " " + line.Container + ": " + line.Message
			}
			select {
			case out <- text:
			case <-ctx.Done():
				// Drain the stream so its producer can finish and exit.
				go func() {
					for range logLines {
					}
					for range errs {
					}
				}()
				return
			}
		}
		// Stopping the command through ctx also ends the stream with an
		// error; only report errors that happened before that.
		if err := <-errs; err != nil && ctx.Err() == nil {
			outErr <- fmt.Errorf("log stream ended: %w", err)
		}
	}()
	return out, outErr, nil
}

// GetFailedDeployLogs returns the logs of containers from the last failed deploy
func (r *DokkuApplicationRepository) GetFailedDeployLogs(ctx context.Context, name *app.ApplicationName) (string, error) {
	output, err := r.dokku.ExecuteCommand(ctx, app.CommandLogsFailed, []string{name.Value()})
	if err != nil {
		return "", fmt.Errorf("failed to get failed deploy logs: %w", err)
	}
	return string(output), nil
}

// SetProcessState restarts, stops or starts all processes of an application
func (r *DokkuApplicationRepository) SetProcessState(ctx context.Context, name *app.ApplicationName, action app.ProcessAction) error {
	var command app.ApplicationCommand
	switch action {
	case app.ProcessRestart:
		command = app.CommandPsRestart
	case app.ProcessStop:
		command = app.CommandPsStop
	case app.ProcessStart:
		command = app.CommandPsStart
	default:
		return fmt.Errorf("unknown process action %q", action)
	}
	if _, err := r.dokku.ExecuteCommand(ctx, command, []string{name.Value()}); err != nil {
		return fmt.Errorf("failed to %s application: %w", action, err)
	}
	return nil
}

// GetDeploySource returns the deploy source and its metadata from apps:report
func (r *DokkuApplicationRepository) GetDeploySource(ctx context.Context, name *app.ApplicationName) (app.DeploySource, error) {
	info, err := r.tryGetBasicApplicationInfo(ctx, name.Value())
	if err != nil {
		return app.DeploySource{}, err
	}
	return app.DeploySource{Type: app.DeploySourceType(info["App deploy source"]), Metadata: info["App deploy source metadata"]}, nil
}

// List retrieves a paginated list of applications
func (r *DokkuApplicationRepository) List(ctx context.Context, offset, limit int) ([]*app.Application, int, error) {
	r.logger.Debug("Retrieving paginated application list",
		"offset", offset,
		"limit", limit)

	allApps, err := r.GetAll(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve all applications: %w", err)
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

// GetByState retrieves applications by state
func (r *DokkuApplicationRepository) GetByState(ctx context.Context, state *app.ApplicationState) ([]*app.Application, error) {
	r.logger.Debug("Retrieving applications by state",
		"state", state.Value())

	allApps, err := r.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve all applications: %w", err)
	}

	var filteredApps []*app.Application
	for _, app := range allApps {
		if app.State().Value() == state.Value() {
			filteredApps = append(filteredApps, app)
		}
	}

	r.logger.Debug("Applications retrieved by state",
		"state", state.Value(),
		"count", len(filteredApps))

	return filteredApps, nil
}

// GetByDomain retrieves applications by domain
func (r *DokkuApplicationRepository) GetByDomain(ctx context.Context, domain string) ([]*app.Application, error) {
	r.logger.Debug("Retrieving applications by domain",
		"domain", domain)

	allApps, err := r.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve all applications: %w", err)
	}

	var filteredApps []*app.Application
	for _, app := range allApps {
		if app.HasDomain(domain) {
			filteredApps = append(filteredApps, app)
		}
	}

	r.logger.Debug("Applications retrieved by domain",
		"domain", domain,
		"count", len(filteredApps))

	return filteredApps, nil
}

// GetRunningApplications retrieves running applications
func (r *DokkuApplicationRepository) GetRunningApplications(ctx context.Context) ([]*app.Application, error) {
	r.logger.Debug("Retrieving running applications")

	runningState, err := app.NewApplicationState(app.StateRunning)
	if err != nil {
		return nil, fmt.Errorf("failed to create running state: %w", err)
	}

	return r.GetByState(ctx, runningState)
}

// CountByState counts applications by state
func (r *DokkuApplicationRepository) CountByState(ctx context.Context) (map[app.StateValue]int, error) {
	r.logger.Debug("Counting applications by state")

	allApps, err := r.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve all applications: %w", err)
	}

	counts := make(map[app.StateValue]int)
	for _, app := range allApps {
		counts[app.State().Value()]++
	}

	r.logger.Debug("Count by state completed",
		"states", len(counts))

	return counts, nil
}

// GetApplicationMetrics retrieves application metrics
func (r *DokkuApplicationRepository) GetApplicationMetrics(ctx context.Context) (*app.ApplicationMetrics, error) {
	r.logger.Debug("Retrieving application metrics")

	allApps, err := r.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve all applications: %w", err)
	}

	counts, err := r.CountByState(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count by state: %w", err)
	}

	metrics := &app.ApplicationMetrics{
		TotalApplications:     len(allApps),
		RunningApplications:   counts[app.StateRunning],
		StoppedApplications:   counts[app.StateStopped],
		ErrorApplications:     counts[app.StateError],
		ApplicationsByState:   counts,
		MostUsedBuildpacks:    make(map[string]int),
		TotalDeployments:      0,
		SuccessfulDeployments: 0,
		FailedDeployments:     0,
		AverageDeploymentTime: 0.0,
	}

	r.logger.Debug("Application metrics retrieved")

	return metrics, nil
}

// GetApplicationsWithBuildpack retrieves applications with a specific buildpack
func (r *DokkuApplicationRepository) GetApplicationsWithBuildpack(ctx context.Context, buildpack string) ([]*app.Application, error) {
	r.logger.Debug("Retrieving applications by buildpack",
		"buildpack", buildpack)

	allApps, err := r.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve all applications: %w", err)
	}

	// For now, return all applications since buildpack detection is not implemented
	// This can be enhanced later when buildpack information is available
	r.logger.Debug("Applications retrieved by buildpack",
		"buildpack", buildpack,
		"count", len(allApps))

	return allApps, nil
}

// GetRecentlyDeployed retrieves recently deployed applications
func (r *DokkuApplicationRepository) GetRecentlyDeployed(ctx context.Context, limit int) ([]*app.Application, error) {
	r.logger.Debug("Retrieving recently deployed applications",
		"limit", limit)

	allApps, err := r.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve all applications: %w", err)
	}

	// For now, return limited number of applications
	// This can be enhanced later when deployment timestamps are available
	if len(allApps) > limit {
		allApps = allApps[:limit]
	}

	r.logger.Debug("Recently deployed applications retrieved",
		"count", len(allApps))

	return allApps, nil
}

// updateApplicationFromInfo updates the application with retrieved information
func (r *DokkuApplicationRepository) updateApplicationFromInfo(app *app.Application, info map[string]string, config map[string]string) error {
	// Apply environment variables
	for key, value := range config {
		if err := app.SetEnvironmentVariable(key, value); err != nil {
			r.logger.Warn("Failed to set environment variable",
				"key", key,
				"error", err)
		}
	}

	// Process processes if present in information
	if processesStr, ok := info["ps.scale"]; ok && processesStr != "" {
		r.parseProcesses(app, processesStr)
	}

	// Process domains if present
	if domainsStr, ok := info["domains"]; ok && domainsStr != "" {
		domains := strings.SplitSeq(domainsStr, " ")
		for domain := range domains {
			if domain != "" {
				if err := app.AddDomain(domain); err != nil {
					r.logger.Warn("Failed to add domain",
						"domain", domain,
						"error", err)
				}
			}
		}
	}

	return nil
}

// parseProcesses parses and adds processes from a string
func (r *DokkuApplicationRepository) parseProcesses(application *app.Application, processesStr string) {
	processes := strings.FieldsSeq(processesStr)
	for proc := range processes {
		parts := strings.Split(proc, ":")
		if len(parts) == 2 {
			processType := parts[0]
			scaleStr := parts[1]

			scale, err := strconv.Atoi(scaleStr)
			if err != nil {
				r.logger.Warn("Failed to parse process scale",
					"process", proc,
					"error", err)
				continue
			}

			processTypeVO, err := process.NewProcessType(processType)
			if err != nil {
				r.logger.Warn("Invalid process type",
					"type", processType,
					"error", err)
				continue
			}

			// Use AddProcessForScaling since we don't have command information from parsing
			if err := application.AddProcessForScaling(processTypeVO, scale); err != nil {
				r.logger.Warn("Failed to add process for scaling",
					"type", processType,
					"error", err)
			}
		}
	}
}

// tryGetPsReportInfo tries to retrieve ps:report information for proper state detection
func (r *DokkuApplicationRepository) tryGetPsReportInfo(ctx context.Context, appName string) (map[string]string, error) {
	output, err := r.dokku.ExecuteCommand(ctx, app.CommandPsReport, []string{appName})
	if err != nil {
		return nil, fmt.Errorf("failed to execute ps:report: %w", err)
	}

	// Parse ps:report output to extract deployment and running state
	info := make(map[string]string)
	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				info[key] = value
			}
		}
	}

	return info, nil
}

// tryGetBasicApplicationInfo tries to retrieve basic information
func (r *DokkuApplicationRepository) tryGetBasicApplicationInfo(ctx context.Context, appName string) (map[string]string, error) {
	output, err := r.dokku.ExecuteCommand(ctx, app.CommandAppsReport, []string{appName})
	if err != nil {
		return nil, fmt.Errorf("failed to execute apps:report: %w", err)
	}

	// Parse apps:report output to extract basic information
	info := make(map[string]string)
	lines := strings.SplitSeq(string(output), "\n")
	for line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				info[key] = value
			}
		}
	}

	return info, nil
}

// determineStateFromInfo determines the application state from Dokku output
func (r *DokkuApplicationRepository) determineStateFromInfo(info map[string]string) app.StateValue {
	// Detect locked applications that have a deploy source but are not fully deployed
	if locked, ok := info["App locked"]; ok && strings.EqualFold(locked, "true") {
		if deploySource, ok := info["App deploy source"]; ok && strings.TrimSpace(deploySource) != "" {
			if deployed, ok := info["Deployed"]; !ok || strings.EqualFold(deployed, "false") {
				return app.StateError
			}
		}
	}

	// Check for deployment status first (from ps:report output)
	if deployed, ok := info["Deployed"]; ok {
		if deployed == "false" {
			return app.StateExists // App exists but is not deployed
		}
	}

	// Check for running status (from ps:report output)
	switch info["Running"] {
	case "true":
		return app.StateRunning
	case "false":
		return app.StateStopped
	}

	// Check for process scale information to determine if app is running (fallback)
	if processesStr, ok := info["ps.scale"]; ok && processesStr != "" {
		// If there are processes with scale > 0, app is likely running
		if r.hasRunningProcesses(processesStr) {
			return app.StateRunning
		}
		// If processes exist but scale is 0, app is stopped
		return app.StateStopped
	}

	// Check app status if available (fallback)
	if status, ok := info["status"]; ok {
		switch status {
		case "running":
			return app.StateRunning
		case "stopped":
			return app.StateStopped
		case "error", "failed":
			return app.StateError
		}
	}

	// Default to exists if we can't determine specific state
	return app.StateExists
}

// hasRunningProcesses checks if any processes have scale > 0
func (r *DokkuApplicationRepository) hasRunningProcesses(processesStr string) bool {
	processes := strings.FieldsSeq(processesStr)
	for proc := range processes {
		parts := strings.Split(proc, ":")
		if len(parts) == 2 {
			if scale, err := strconv.Atoi(parts[1]); err == nil && scale > 0 {
				return true
			}
		}
	}
	return false
}

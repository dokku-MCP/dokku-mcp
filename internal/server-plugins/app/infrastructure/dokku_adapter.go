package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	dokkuApi "github.com/alex-galey/dokku-mcp/internal/dokku-api"
	app "github.com/alex-galey/dokku-mcp/internal/server-plugins/app/domain"
)

// DokkuApplicationAdapter provides application-specific operations using the generic DokkuClient
// This adapter encapsulates all application-related Dokku command logic
type DokkuApplicationAdapter struct {
	client dokkuApi.DokkuClient
	logger *slog.Logger
}

// NewDokkuApplicationAdapter creates a new application adapter
func NewDokkuApplicationAdapter(client dokkuApi.DokkuClient, logger *slog.Logger) *DokkuApplicationAdapter {
	return &DokkuApplicationAdapter{
		client: client,
		logger: logger,
	}
}

// ExecuteCommand wraps the client's ExecuteCommand with application-specific context and validation
func (a *DokkuApplicationAdapter) ExecuteCommand(ctx context.Context, command app.ApplicationCommand, args []string) ([]byte, error) {
	// Validate command is allowed
	if !command.IsValid() {
		return nil, fmt.Errorf("invalid application command: %s", command)
	}

	return a.client.ExecuteCommand(ctx, command.String(), args)
}

// GetApplications retrieves list of all applications
func (a *DokkuApplicationAdapter) GetApplications(ctx context.Context) ([]string, error) {
	output, err := a.ExecuteCommand(ctx, app.CommandAppsList, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to get applications: %w", err)
	}

	a.logger.Debug("Raw output from dokku apps:list",
		"output", string(output),
		"output_len", len(output))

	apps := dokkuApi.ParseLinesSkipHeaders(string(output))

	a.logger.Debug("Applications retrieved",
		"count", len(apps),
		"apps", apps)

	return apps, nil
}

// GetApplicationInfo retrieves detailed information about an application
func (a *DokkuApplicationAdapter) GetApplicationInfo(ctx context.Context, appName string) (map[string]string, error) {
	output, err := a.ExecuteCommand(ctx, app.CommandAppsInfo, []string{appName})
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			a.logger.Debug("apps:info returned exit status 1 - application probably not deployed",
				"app_name", appName,
				"suggestion", "application exists but has no detailed information available")
		}
		return nil, fmt.Errorf("failed to get application info %s: %w", appName, err)
	}

	info := dokkuApi.ParseKeyValueOutput(string(output), ":")
	return info, nil
}

// GetApplicationConfig retrieves application configuration
func (a *DokkuApplicationAdapter) GetApplicationConfig(ctx context.Context, appName string) (map[string]string, error) {
	output, err := a.ExecuteCommand(ctx, app.CommandConfigShow, []string{appName})
	if err != nil {
		return nil, fmt.Errorf("failed to get application config %s: %w", appName, err)
	}

	config := dokkuApi.ParseKeyValueOutput(string(output), "=")
	return config, nil
}

// SetApplicationConfig sets application configuration
func (a *DokkuApplicationAdapter) SetApplicationConfig(ctx context.Context, appName string, config map[string]string) error {
	var args []string
	args = append(args, appName)

	for key, value := range config {
		args = append(args, fmt.Sprintf("%s=%s", key, value))
	}

	_, err := a.ExecuteCommand(ctx, app.CommandConfigSet, args)
	if err != nil {
		return fmt.Errorf("failed to set application config %s: %w", appName, err)
	}

	return nil
}

// ScaleApplication scales application processes
func (a *DokkuApplicationAdapter) ScaleApplication(ctx context.Context, appName string, processType string, count int) error {
	scaleArg := fmt.Sprintf("%s=%d", processType, count)
	_, err := a.ExecuteCommand(ctx, app.CommandPsScale, []string{appName, scaleArg})
	if err != nil {
		return fmt.Errorf("failed to scale application %s: %w", appName, err)
	}

	return nil
}

// GetApplicationLogs retrieves application logs
func (a *DokkuApplicationAdapter) GetApplicationLogs(ctx context.Context, appName string, lines int) (string, error) {
	args := []string{appName}
	if lines > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", lines))
	}

	output, err := a.ExecuteCommand(ctx, app.CommandLogs, args)
	if err != nil {
		return "", fmt.Errorf("failed to get application logs %s: %w", appName, err)
	}

	return string(output), nil
}

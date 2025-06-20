package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	dokkuApi "github.com/alex-galey/dokku-mcp/internal/dokku-api"
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

// ExecuteCommand wraps the client's ExecuteCommand with application-specific context
func (a *DokkuApplicationAdapter) ExecuteCommand(ctx context.Context, command string, args []string) ([]byte, error) {
	return a.client.ExecuteCommand(ctx, command, args)
}

// GetApplications retrieves list of all applications
func (a *DokkuApplicationAdapter) GetApplications(ctx context.Context) ([]string, error) {
	output, err := a.ExecuteCommand(ctx, "apps:list", []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to get applications: %w", err)
	}

	// Log de debug pour voir la sortie brute
	a.logger.Debug("Sortie brute de dokku apps:list",
		"output", string(output),
		"output_len", len(output))

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var apps []string

	// Log de debug pour voir chaque ligne
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		a.logger.Debug("Traitement ligne",
			"index", i,
			"line_raw", line,
			"line_trimmed", trimmedLine,
			"starts_with_equals", strings.HasPrefix(trimmedLine, "===="),
			"is_empty", trimmedLine == "")

		if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "====") {
			apps = append(apps, trimmedLine)
			a.logger.Debug("Application trouvée", "app_name", trimmedLine)
		}
	}

	a.logger.Debug("Applications retrieved",
		"count", len(apps),
		"apps", apps)

	return apps, nil
}

// GetApplicationInfo retrieves detailed information about an application
func (a *DokkuApplicationAdapter) GetApplicationInfo(ctx context.Context, appName string) (map[string]interface{}, error) {
	output, err := a.ExecuteCommand(ctx, "apps:info", []string{appName})
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			a.logger.Debug("apps:info returned exit status 1 - application probably not deployed",
				"app_name", appName,
				"suggestion", "application exists but has no detailed information available")
		}
		return nil, fmt.Errorf("failed to get application info %s: %w", appName, err)
	}

	// Use the common function to parse key:value pairs
	stringMap := a.parseOutputLines(output, ":")

	info := make(map[string]any)
	for key, value := range stringMap {
		info[key] = value
	}

	return info, nil
}

// GetApplicationConfig retrieves application configuration
func (a *DokkuApplicationAdapter) GetApplicationConfig(ctx context.Context, appName string) (map[string]string, error) {
	output, err := a.ExecuteCommand(ctx, "config:show", []string{appName})
	if err != nil {
		return nil, fmt.Errorf("failed to get application config %s: %w", appName, err)
	}

	config := a.parseOutputLines(output, "=")
	return config, nil
}

// SetApplicationConfig sets application configuration
func (a *DokkuApplicationAdapter) SetApplicationConfig(ctx context.Context, appName string, config map[string]string) error {
	var args []string
	args = append(args, appName)

	for key, value := range config {
		args = append(args, fmt.Sprintf("%s=%s", key, value))
	}

	_, err := a.ExecuteCommand(ctx, "config:set", args)
	if err != nil {
		return fmt.Errorf("failed to set application config %s: %w", appName, err)
	}

	return nil
}

// ScaleApplication scales application processes
func (a *DokkuApplicationAdapter) ScaleApplication(ctx context.Context, appName string, processType string, count int) error {
	scaleArg := fmt.Sprintf("%s=%d", processType, count)
	_, err := a.ExecuteCommand(ctx, "ps:scale", []string{appName, scaleArg})
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

	output, err := a.ExecuteCommand(ctx, "logs", args)
	if err != nil {
		return "", fmt.Errorf("failed to get application logs %s: %w", appName, err)
	}

	return string(output), nil
}

// parseOutputLines parses command output into key-value pairs
func (a *DokkuApplicationAdapter) parseOutputLines(output []byte, separator string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.Contains(line, separator) {
			parts := strings.SplitN(line, separator, 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				result[key] = value
			}
		}
	}

	return result
}

package dokku

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/alex-galey/dokku-mcp/internal/domain"
)

// pluginDiscoveryService implements the domain.PluginDiscoveryService interface
// using the Dokku CLI infrastructure.
type pluginDiscoveryService struct {
	client DokkuClient
	logger *slog.Logger
}

// NewPluginDiscoveryService creates a new plugin discovery service.
func NewPluginDiscoveryService(client DokkuClient, logger *slog.Logger) domain.PluginDiscoveryService {
	return &pluginDiscoveryService{
		client: client,
		logger: logger,
	}
}

// GetEnabledPlugins retrieves the list of enabled Dokku plugins.
func (s *pluginDiscoveryService) GetEnabledPlugins(ctx context.Context) ([]string, error) {
	// Create a context with shorter timeout for plugin detection
	pluginCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Execute 'dokku plugin:list' command
	output, err := s.client.ExecuteCommand(pluginCtx, "plugin:list", []string{})
	if err != nil {
		s.logger.Warn("Failed to execute plugin:list command, assuming no plugins are enabled. This is normal in development environments without Dokku.",
			"error", err)
		// Return empty slice instead of error - core plugins should still work
		return []string{}, nil
	}

	// Parse the output to extract enabled plugin names
	var enabledPlugins []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "====") {
			continue
		}

		// Plugin list format: "plugin-name    version    enabled/disabled"
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			pluginName := parts[0]
			status := parts[len(parts)-1]

			if status == "enabled" {
				enabledPlugins = append(enabledPlugins, pluginName)
			}
		}
	}

	s.logger.Debug("Successfully retrieved enabled Dokku plugins",
		"plugins", enabledPlugins,
		"count", len(enabledPlugins))

	return enabledPlugins, nil
}

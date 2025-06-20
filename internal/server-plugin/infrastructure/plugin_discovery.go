package infrastructure

import (
	"context"
	"log/slog"
	"strings"
	"time"

	dokkuApi "github.com/alex-galey/dokku-mcp/internal/dokku-api"
	"github.com/alex-galey/dokku-mcp/internal/server-plugin/domain"
)

// srvPluginDiscoveryService implements the domain.PluginDiscoveryService interface
// using the Dokku CLI infrastructure.
type srvPluginDiscoveryService struct {
	client dokkuApi.DokkuClient
	logger *slog.Logger
}

// NewPluginDiscoveryService creates a new plugin discovery service.
func NewPluginDiscoveryService(client dokkuApi.DokkuClient, logger *slog.Logger) domain.ServerPluginDiscoveryService {
	return &srvPluginDiscoveryService{
		client: client,
		logger: logger,
	}
}

// GetEnabledDokkuPlugins retrieves the list of enabled Dokku plugins.
func (s *srvPluginDiscoveryService) GetEnabledDokkuPlugins(ctx context.Context) ([]string, error) {
	pluginCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Execute 'dokku plugin:list' command - no server-plugin plugin/ is loaded yet at this stage
	output, err := s.client.ExecuteCommand(pluginCtx, "plugin:list", []string{})
	if err != nil {
		s.logger.Error("Failed to execute plugin:list",
			"error", err)
		return nil, err
	}

	// Parse the output to extract enabled plugin names
	var enabledPlugins []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "====") {
			continue
		}

		// Plugin list format: "plugin-name    version    enabled/disabled    description..."
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			pluginName := parts[0]
			status := parts[2] // Status is in 3rd position: name version status

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

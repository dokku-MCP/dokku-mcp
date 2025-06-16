package plugins

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/alex-galey/dokku-mcp/internal/domain"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/fx"
)

// DynamicPluginRegistry manages the lifecycle of FeaturePlugins based on
// the availability of Dokku plugins. It provides dynamic activation and
// deactivation of plugins without requiring server restarts.
type DynamicPluginRegistry struct {
	mcpServer       *server.MCPServer
	pluginDiscovery domain.PluginDiscoveryService
	logger          *slog.Logger

	// allPlugins holds all registered plugins injected by Fx
	allPlugins []domain.FeaturePlugin

	// active tracks which plugins are currently active
	active map[string]bool

	// mu protects concurrent access to the registry state
	mu sync.RWMutex

	// syncInterval defines how often to check for plugin changes
	syncInterval time.Duration
}

// NewDynamicPluginRegistry creates a new dynamic plugin registry.
// Fx automatically provides the logger, plugin discovery service, and a slice of all registered FeaturePlugins.
func NewDynamicPluginRegistry(
	mcpServer *server.MCPServer,
	pluginDiscovery domain.PluginDiscoveryService,
	logger *slog.Logger,
	plugins []domain.FeaturePlugin, // Injected by Fx
) *DynamicPluginRegistry {
	return &DynamicPluginRegistry{
		mcpServer:       mcpServer,
		pluginDiscovery: pluginDiscovery,
		logger:          logger,
		allPlugins:      plugins,
		active:          make(map[string]bool),
		syncInterval:    60 * time.Second, // Default sync every minute
	}
}

// RegisterHooks connects the registry's lifecycle to the Fx application lifecycle.
func (r *DynamicPluginRegistry) RegisterHooks(lc fx.Lifecycle) {
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			r.logger.Info("DynamicPluginRegistry starting...")

			for _, plugin := range r.allPlugins {
				r.logger.Debug("Registering plugin",
					"plugin", plugin.Name(),
					"dokku_plugin", plugin.DokkuPluginName())
			}

			// Start background sync loop
			go r.runSyncLoop(ctx, r.syncInterval)
			return nil
		},
		OnStop: func(context.Context) error {
			r.logger.Info("DynamicPluginRegistry stopping...")
			cancel()
			return nil
		},
	})
}

// runSyncLoop runs the periodic synchronization in a background goroutine.
func (r *DynamicPluginRegistry) runSyncLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Plugin synchronization loop stopped")
			return
		case <-ticker.C:
			if err := r.syncPlugins(ctx); err != nil {
				r.logger.Error("Plugin sync failed", "error", err)
			}
		}
	}
}

// syncPlugins performs the actual synchronization by checking Dokku plugins
// and activating/deactivating plugins as needed.
func (r *DynamicPluginRegistry) syncPlugins(ctx context.Context) error {
	r.logger.Debug("Starting plugin synchronization")

	// Get list of enabled Dokku plugins (with graceful error handling)
	enabledPlugins, err := r.pluginDiscovery.GetEnabledPlugins(ctx)
	if err != nil {
		r.logger.Error("Failed to get enabled Dokku plugins, proceeding with core plugins only", "error", err)
		enabledPlugins = []string{} // Empty list - only core plugins will be activated
	}

	r.logger.Debug("Enabled Dokku plugins detected",
		"plugins", enabledPlugins)

	r.mu.Lock()
	defer r.mu.Unlock()

	activatedCount := 0
	deactivatedCount := 0

	// Check each registered plugin
	for _, plugin := range r.allPlugins {
		pluginName := plugin.Name()
		dokkuPluginName := plugin.DokkuPluginName()

		// Core plugins (empty dokkuPlugin name) are always activated
		// Other plugins are activated only if their dokkuPlugin is enabled
		shouldBeActive := dokkuPluginName == "" || r.isPluginEnabled(dokkuPluginName, enabledPlugins)
		isCurrentlyActive := r.active[pluginName]

		r.logger.Debug("Plugin activation check",
			"plugin", pluginName,
			"dokkuPlugin", dokkuPluginName,
			"should_be_active", shouldBeActive,
			"currently_active", isCurrentlyActive)

		if shouldBeActive && !isCurrentlyActive {
			// Activate plugin
			if err := r.activatePlugin(plugin); err != nil {
				r.logger.Error("Failed to activate plugin",
					"plugin", pluginName,
					"error", err)
				continue
			}
			r.active[pluginName] = true
			r.logger.Info("Plugin activated",
				"plugin", pluginName,
				"dokkuPlugin", dokkuPluginName)
			activatedCount++

		} else if !shouldBeActive && isCurrentlyActive {
			// Deactivate plugin
			if err := r.deactivatePlugin(plugin); err != nil {
				r.logger.Error("Failed to deactivate plugin",
					"plugin", pluginName,
					"error", err)
				continue
			}
			r.active[pluginName] = false
			r.logger.Info("Plugin deactivated",
				"plugin", pluginName,
				"dokkuPlugin", dokkuPluginName)
			deactivatedCount++
		}
	}

	r.logger.Info("Plugin synchronization completed",
		"activated", activatedCount,
		"deactivated", deactivatedCount,
		"total_active", r.getActivePluginsCountUnsafe())

	return nil
}

// getActivePluginsCountUnsafe returns the count of active plugins without acquiring a lock.
// This method should only be called when the caller already holds the lock.
func (r *DynamicPluginRegistry) getActivePluginsCountUnsafe() int {
	count := 0
	for _, plugin := range r.allPlugins {
		if r.active[plugin.Name()] {
			count++
		}
	}
	return count
}

// GetActivePlugins returns a list of currently active plugins.
func (r *DynamicPluginRegistry) GetActivePlugins() []domain.FeaturePlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var activePlugins []domain.FeaturePlugin
	for _, plugin := range r.allPlugins {
		if r.active[plugin.Name()] {
			activePlugins = append(activePlugins, plugin)
		}
	}

	return activePlugins
}

// IsPluginActive checks if a specific plugin is currently active.
func (r *DynamicPluginRegistry) IsPluginActive(pluginName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.active[pluginName]
}

// isPluginEnabled checks if a plugin is in the list of enabled plugins.
func (r *DynamicPluginRegistry) isPluginEnabled(pluginName string, enabledPlugins []string) bool {
	for _, enabled := range enabledPlugins {
		if enabled == pluginName {
			return true
		}
	}
	return false
}

// activatePlugin registers a plugin's resources and tools with the MCP server.
func (r *DynamicPluginRegistry) activatePlugin(plugin domain.FeaturePlugin) error {
	if err := plugin.Register(r.mcpServer); err != nil {
		return fmt.Errorf("failed to register plugin %s: %w", plugin.Name(), err)
	}
	return nil
}

// deactivatePlugin removes a plugin's resources and tools from the MCP server.
func (r *DynamicPluginRegistry) deactivatePlugin(plugin domain.FeaturePlugin) error {
	if err := plugin.Deregister(r.mcpServer); err != nil {
		return fmt.Errorf("failed to deregister plugin %s: %w", plugin.Name(), err)
	}
	return nil
}

// SyncPlugins performs a manual synchronization of plugins (exposed for testing).
func (r *DynamicPluginRegistry) SyncPlugins(ctx context.Context) error {
	return r.syncPlugins(ctx)
}

package plugins

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/alex-galey/dokku-mcp/internal/server-plugin/domain"
	"github.com/alex-galey/dokku-mcp/pkg/config"
	"go.uber.org/fx"
)

// ServerPluginRegistry manages the basic registration of server plugins
type ServerPluginRegistry struct {
	plugins map[string]domain.ServerPlugin
	mu      sync.RWMutex
}

// NewServerPluginRegistry creates a new server plugin registry
func NewServerPluginRegistry() *ServerPluginRegistry {
	return &ServerPluginRegistry{
		plugins: make(map[string]domain.ServerPlugin),
	}
}

// Register registers a server plugin
func (r *ServerPluginRegistry) Register(plugin domain.ServerPlugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.plugins[plugin.ID()] = plugin
	return nil
}

// GetResourceProviders returns all plugins that provide resources
func (r *ServerPluginRegistry) GetResourceProviders() []domain.ResourceProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var providers []domain.ResourceProvider
	for _, plugin := range r.plugins {
		if provider, ok := plugin.(domain.ResourceProvider); ok {
			providers = append(providers, provider)
		}
	}
	return providers
}

// GetToolProviders returns all plugins that provide tools
func (r *ServerPluginRegistry) GetToolProviders() []domain.ToolProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var providers []domain.ToolProvider
	for _, plugin := range r.plugins {
		if provider, ok := plugin.(domain.ToolProvider); ok {
			providers = append(providers, provider)
		}
	}
	return providers
}

// GetPromptProviders returns all plugins that provide prompts
func (r *ServerPluginRegistry) GetPromptProviders() []domain.PromptProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var providers []domain.PromptProvider
	for _, plugin := range r.plugins {
		if provider, ok := plugin.(domain.PromptProvider); ok {
			providers = append(providers, provider)
		}
	}
	return providers
}

// ServerPluginProvider interface that matches what MCPAdapter expects
type ServerPluginProvider interface {
	GetResourceProviders() []domain.ResourceProvider
	GetToolProviders() []domain.ToolProvider
	GetPromptProviders() []domain.PromptProvider
}

// DynamicServerPluginRegistry manages the lifecycle of server plugins based on
// the availability of Dokku server plugins.
type DynamicServerPluginRegistry struct {
	pluginRegistry  *ServerPluginRegistry // Our own plugin registry
	pluginDiscovery domain.ServerPluginDiscoveryService
	logger          *slog.Logger
	srvConfig       *config.ServerConfig

	allServerPlugins []domain.ServerPlugin
	active           map[string]bool
	mu               sync.RWMutex
}

type DynamicServerPluginRegistryParams struct {
	fx.In
	PluginRegistry  *ServerPluginRegistry
	PluginDiscovery domain.ServerPluginDiscoveryService
	Logger          *slog.Logger
	SrvConfig       *config.ServerConfig
	ServerPlugins   []domain.ServerPlugin `group:"server_plugins"`
}

// NewDynamicServerPluginRegistry creates a new dynamic server plugin registry
func NewDynamicServerPluginRegistry(params DynamicServerPluginRegistryParams) *DynamicServerPluginRegistry {
	return &DynamicServerPluginRegistry{
		pluginRegistry:   params.PluginRegistry,
		pluginDiscovery:  params.PluginDiscovery,
		logger:           params.Logger,
		srvConfig:        params.SrvConfig,
		allServerPlugins: params.ServerPlugins,
		active:           make(map[string]bool),
	}
}

// RegisterHooks connects the registry's lifecycle to the Fx application lifecycle.
func (r *DynamicServerPluginRegistry) RegisterHooks(lc fx.Lifecycle) {
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			r.logger.Info("DynamicServerPluginRegistry starting...")

			// Register all server plugins with the plugin registry first
			for _, srvPlugin := range r.allServerPlugins {
				if err := r.pluginRegistry.Register(srvPlugin); err != nil {
					r.logger.Error("Failed to register server plugin",
						"plugin", srvPlugin.ID(),
						"error", err)
					continue
				}
				r.logger.Debug("ServerPlugin registered with registry",
					"plugin", srvPlugin.ID(),
					"name", srvPlugin.Name(),
					"dokku_plugin", srvPlugin.DokkuPluginName())
			}

			// Start background sync loop only if enabled and interval > 0
			if r.srvConfig.PluginDiscovery.Enabled && r.srvConfig.PluginDiscovery.SyncInterval > 0 {
				r.logger.Info("Starting plugin discovery sync loop",
					"interval", r.srvConfig.PluginDiscovery.SyncInterval)
				go r.runSyncLoop(ctx, r.srvConfig.PluginDiscovery.SyncInterval)
			} else {
				r.logger.Info("Plugin discovery sync loop disabled")
				// NOTE: Initial sync is now handled by server hooks before MCP registration
				// This prevents the timing issue where resources/prompts are not available initially
			}
			return nil
		},
		OnStop: func(context.Context) error {
			r.logger.Info("DynamicServerPluginRegistry stopping...")
			cancel()
			return nil
		},
	})
}

// runSyncLoop runs the periodic synchronization in a background goroutine.
func (r *DynamicServerPluginRegistry) runSyncLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("ServerPlugin synchronization loop stopped")
			return
		case <-ticker.C:
			if err := r.syncServerPlugins(ctx); err != nil {
				r.logger.Error("ServerPlugin sync failed", "error", err)
			}
		}
	}
}

// syncServerPlugins performs the actual synchronization by checking Dokku plugins
// and activating/deactivating server plugins as needed.
func (r *DynamicServerPluginRegistry) syncServerPlugins(ctx context.Context) error {
	r.logger.Debug("Starting server plugin synchronization")

	// Get list of enabled Dokku plugins (with graceful error handling)
	enabledDokkuPlugins, err := r.pluginDiscovery.GetEnabledDokkuPlugins(ctx)
	if err != nil {
		r.logger.Error("Failed to get enabled Dokku plugins, proceeding with core plugins only", "error", err)
		enabledDokkuPlugins = []string{} // Empty list - only core server plugins will be activated
	}

	r.logger.Debug("Enabled Dokku plugins detected", "plugins", enabledDokkuPlugins)

	r.mu.Lock()
	defer r.mu.Unlock()

	activatedCount := 0
	deactivatedCount := 0

	// Check each registered server plugin
	for _, srvPlugin := range r.allServerPlugins {
		srvPluginID := srvPlugin.ID()
		dokkuPluginName := srvPlugin.DokkuPluginName()

		// Core server plugins (empty dokkuPlugin name) are always activated
		// Other server plugins are activated only if their dokkuPlugin is enabled
		shouldBeActive := dokkuPluginName == "" || r.isDokkuPluginEnabled(dokkuPluginName, enabledDokkuPlugins)
		isCurrentlyActive := r.active[srvPluginID]

		r.logger.Debug("ServerPlugin activation check",
			"plugin", srvPluginID,
			"name", srvPlugin.Name(),
			"dokku_plugin", dokkuPluginName,
			"should_be_active", shouldBeActive,
			"currently_active", isCurrentlyActive)

		if shouldBeActive && !isCurrentlyActive {
			// Activate server plugin - just mark as active, MCP registration happens separately
			r.active[srvPluginID] = true
			r.logger.Info("ServerPlugin activated",
				"plugin", srvPluginID,
				"name", srvPlugin.Name(),
				"dokku_plugin", dokkuPluginName)
			activatedCount++

		} else if !shouldBeActive && isCurrentlyActive {
			// Deactivate server plugin - just mark as inactive
			r.active[srvPluginID] = false
			r.logger.Info("ServerPlugin deactivated",
				"plugin", srvPluginID,
				"name", srvPlugin.Name(),
				"dokku_plugin", dokkuPluginName)
			deactivatedCount++
		}
	}

	r.logger.Info("ServerPlugin synchronization completed",
		"activated", activatedCount,
		"deactivated", deactivatedCount,
		"total_active", r.getActiveServerPluginsCountUnsafe())

	return nil
}

// getActiveServerPluginsCountUnsafe returns the count of active server plugins without acquiring a lock.
// This method should only be called when the caller already holds the lock.
func (r *DynamicServerPluginRegistry) getActiveServerPluginsCountUnsafe() int {
	count := 0
	for _, plugin := range r.allServerPlugins {
		if r.active[plugin.ID()] {
			count++
		}
	}
	return count
}

// GetActiveServerPlugins returns a list of currently active server plugins.
func (r *DynamicServerPluginRegistry) GetActiveServerPlugins() []domain.ServerPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var activeServerPlugins []domain.ServerPlugin
	for _, srvPlugin := range r.allServerPlugins {
		if r.active[srvPlugin.ID()] {
			activeServerPlugins = append(activeServerPlugins, srvPlugin)
		}
	}

	return activeServerPlugins
}

// IsServerPluginActive checks if a specific plugin is currently active.
func (r *DynamicServerPluginRegistry) IsServerPluginActive(srvPluginID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.active[srvPluginID]
}

// isDokkuPluginEnabled checks if a plugin is in the list of enabled Dokku plugins.
func (r *DynamicServerPluginRegistry) isDokkuPluginEnabled(dokkuPluginName string, enabledDokkuPlugins []string) bool {
	for _, enabled := range enabledDokkuPlugins {
		if enabled == dokkuPluginName {
			return true
		}
	}
	return false
}

// SyncServerPlugins performs a manual synchronization of server plugins (exposed for testing).
func (r *DynamicServerPluginRegistry) SyncServerPlugins(ctx context.Context) error {
	return r.syncServerPlugins(ctx)
}

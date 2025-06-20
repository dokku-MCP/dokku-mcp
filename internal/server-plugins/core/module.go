package core

import (
	"log/slog"

	serverDomain "github.com/alex-galey/dokku-mcp/internal/server-plugin/domain"
	"go.uber.org/fx"
)

// CoreModule provides dependency injection for the core plugin
var CoreModule = fx.Module("core",
	fx.Provide(
		fx.Annotate(
			NewCoreServerPlugin,
			fx.As(new(serverDomain.ServerPlugin)),
			fx.ResultTags(`group:"server_plugins"`),
		),
	),
)

// RegisterCorePlugin registers the core plugin with the server plugin registry
func RegisterCorePlugin(
	plugin serverDomain.ServerPlugin,
	registry interface{}, // Registry interface to avoid import cycles
	logger *slog.Logger,
) error {
	logger.Info("Registering core plugin", "plugin_id", plugin.ID())

	// Type assertion to get the registry Register method
	if registrar, ok := registry.(interface {
		Register(serverDomain.ServerPlugin) error
	}); ok {
		return registrar.Register(plugin)
	}

	logger.Warn("Registry does not support plugin registration")
	return nil
}

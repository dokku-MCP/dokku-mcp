package core

import (
	"log/slog"

	serverDomain "github.com/alex-galey/dokku-mcp/internal/server-plugin/domain"
	"go.uber.org/fx"
)

// PluginRegistry defines the interface for registering server plugins
type PluginRegistry interface {
	Register(plugin serverDomain.ServerPlugin) error
}

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
	registry PluginRegistry,
	logger *slog.Logger,
) error {
	logger.Info("Registering core plugin", "plugin_id", plugin.ID())
	return registry.Register(plugin)
}

package onboarding

import (
	"context"
	"log/slog"

	mcpserver "github.com/dokku-mcp/dokku-mcp/internal/server"
	serverDomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	"go.uber.org/fx"
)

var Module = fx.Module("onboarding",
	fx.Provide(
		// Provide the concrete plugin for internal injection (SetProvider in Invoke)
		NewOnboardingServerPlugin,
		// Also expose it as a grouped ServerPlugin for registration
		fx.Annotate(
			func(p *OnboardingServerPlugin) serverDomain.ServerPlugin { return p },
			fx.As(new(serverDomain.ServerPlugin)),
			fx.ResultTags(`group:"server_plugins"`),
		),
	),
	fx.Invoke(func(lc fx.Lifecycle, logger *slog.Logger, adapter mcpserver.ServerPluginProvider, p *OnboardingServerPlugin) {
		lc.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				p.SetProvider(adapter)
				logger.Info("Onboarding plugin initialized")
				return nil
			},
		})
	}),
)

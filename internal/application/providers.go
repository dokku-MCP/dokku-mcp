package application

import (
	"github.com/alex-galey/dokku-mcp/internal/application/dto"
	"github.com/alex-galey/dokku-mcp/internal/application/handlers"
	"github.com/alex-galey/dokku-mcp/internal/application/plugins"
	"github.com/alex-galey/dokku-mcp/internal/application/services"
	"github.com/alex-galey/dokku-mcp/internal/application/usecases"
	"github.com/alex-galey/dokku-mcp/internal/application/workflows"
	"github.com/alex-galey/dokku-mcp/internal/domain"
	"go.uber.org/fx"
)

// ApplicationProviders provides all application layer dependencies
var ApplicationProviders = fx.Options(
	// Application services
	fx.Provide(
		services.NewApplicationDeploymentService,
		usecases.NewApplicationUseCase,
		dto.NewApplicationMapper,
	),

	// MCP handlers
	fx.Provide(
		fx.Annotate(
			handlers.NewMCPApplicationHandler,
			fx.As(new(domain.MCPHandler)),
		),
	),

	// Workflow orchestration
	fx.Provide(
		workflows.NewWorkflowEngine,
	),

	// Plugin system
	fx.Provide(
		// Plugin collector
		fx.Annotate(
			func(plugins ...domain.FeaturePlugin) []domain.FeaturePlugin {
				return plugins
			},
			fx.ParamTags(`group:"feature_plugins"`),
		),

		// Plugin registry
		plugins.NewDynamicPluginRegistry,
	),
)

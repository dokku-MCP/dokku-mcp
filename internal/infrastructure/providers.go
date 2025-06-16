package infrastructure

import (
	"github.com/alex-galey/dokku-mcp/internal/domain"
	"github.com/alex-galey/dokku-mcp/internal/infrastructure/config"
	"github.com/alex-galey/dokku-mcp/internal/infrastructure/dokku"
	"github.com/alex-galey/dokku-mcp/internal/infrastructure/server"
	"github.com/alex-galey/dokku-mcp/internal/infrastructure/workflows"
	"go.uber.org/fx"
)

// InfrastructureProviders provides all infrastructure layer dependencies
var InfrastructureProviders = fx.Options(
	// Core infrastructure
	fx.Provide(
		config.LoadConfig,
		server.NewSlogLogger,
		server.NewMCPServerInstance,
	),

	// Dokku infrastructure
	fx.Provide(
		dokku.NewDokkuClientFromConfig,
		dokku.NewApplicationRepository,
		dokku.NewDeploymentInfrastructure,
		dokku.NewDeploymentRepository,
		fx.Annotate(
			dokku.NewPluginDiscoveryService,
			fx.As(new(domain.PluginDiscoveryService)),
		),
	),

	// Workflow infrastructure
	fx.Provide(
		workflows.ProvideYAMLWorkflowProvider,
	),

	// Feature plugins
	fx.Provide(
		fx.Annotate(
			server.NewCorePlugin,
			fx.As(new(domain.FeaturePlugin)),
			fx.ResultTags(`group:"feature_plugins"`),
		),
	),
)

package deployment

import (
	"log/slog"
	"time"

	dokkuApi "github.com/dokku-mcp/dokku-mcp/internal/dokku-api"
	server_plugin_domain "github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/deployment/adapter"
	deployment_domain "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/deployment/domain"
	deployment_infrastructure "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/deployment/infrastructure"
	"github.com/dokku-mcp/dokku-mcp/internal/shared"
	"go.uber.org/fx"
)

var Module = fx.Module("deployment",
	fx.Provide(
		// Deployment repository
		fx.Annotate(
			func(logger *slog.Logger) deployment_domain.DeploymentRepository {
				return deployment_infrastructure.NewDeploymentRepository(logger)
			},
		),
		// Deployment tracker
		fx.Annotate(
			deployment_domain.NewDeploymentTracker,
		),
		// Deployment status checker
		fx.Annotate(
			func(client dokkuApi.DokkuClient) deployment_domain.DeploymentStatusChecker {
				return deployment_infrastructure.NewDeploymentStatusChecker(client)
			},
		),
		// Deployment poller
		fx.Annotate(
			func(
				tracker *deployment_domain.DeploymentTracker,
				statusChecker deployment_domain.DeploymentStatusChecker,
				logger *slog.Logger,
			) *deployment_domain.DeploymentPoller {
				return deployment_domain.NewDeploymentPoller(
					tracker,
					statusChecker,
					logger,
					10*time.Second, // Poll every 10 seconds
					30*time.Minute, // Max 30 minutes for deployment
				)
			},
		),
		// Deployment infrastructure
		fx.Annotate(
			deployment_infrastructure.NewDeploymentInfrastructure,
		),
		// Deployment service
		fx.Annotate(
			deployment_domain.NewApplicationDeploymentService,
			fx.As(new(deployment_domain.DeploymentService)),
		),
		// Deployment adapter
		fx.Annotate(
			adapter.NewDeploymentServiceAdapter,
			fx.As(new(shared.DeploymentService)),
		),
		// Deployment server plugin
		fx.Annotate(
			NewDeploymentServerPlugin,
			fx.As(new(server_plugin_domain.ServerPlugin)),
			fx.ResultTags(`group:"server_plugins"`),
		),
	),
)

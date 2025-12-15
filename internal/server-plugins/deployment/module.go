package deployment

import (
	"log/slog"
	"time"

	dokkuApi "github.com/dokku-mcp/dokku-mcp/internal/dokku-api"
	serverPluginDomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/deployment/adapter"
	deploymentDomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/deployment/domain"
	deploymentInfrastructure "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/deployment/infrastructure"
	"github.com/dokku-mcp/dokku-mcp/internal/shared"
	"go.uber.org/fx"
)

var Module = fx.Module("deployment",
	fx.Provide(
		// Deployment repository
		fx.Annotate(
			func(logger *slog.Logger) deploymentDomain.DeploymentRepository {
				return deploymentInfrastructure.NewDeploymentRepository(logger)
			},
		),
		// Deployment tracker
		fx.Annotate(
			deploymentDomain.NewDeploymentTracker,
		),
		// Deployment status checker
		fx.Annotate(
			func(client dokkuApi.DokkuClient) deploymentDomain.DeploymentStatusChecker {
				return deploymentInfrastructure.NewDeploymentStatusChecker(client)
			},
		),
		// Deployment poller
		fx.Annotate(
			func(
				tracker *deploymentDomain.DeploymentTracker,
				statusChecker deploymentDomain.DeploymentStatusChecker,
				logger *slog.Logger,
			) *deploymentDomain.DeploymentPoller {
				return deploymentDomain.NewDeploymentPoller(
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
			deploymentInfrastructure.NewDeploymentInfrastructure,
		),
		// Deployment service
		fx.Annotate(
			deploymentDomain.NewApplicationDeploymentService,
			fx.As(new(deploymentDomain.DeploymentService)),
		),
		// Deployment adapter
		fx.Annotate(
			adapter.NewDeploymentServiceAdapter,
			fx.As(new(shared.DeploymentService)),
		),
		// Deployment server plugin
		fx.Annotate(
			NewDeploymentServerPlugin,
			fx.As(new(serverPluginDomain.ServerPlugin)),
			fx.ResultTags(`group:"server_plugins"`),
		),
	),
)

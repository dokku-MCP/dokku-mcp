package deployment

import (
	"log/slog"
	"time"

	dokkuApi "github.com/alex-galey/dokku-mcp/internal/dokku-api"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/deployment/adapter"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/deployment/domain"
	deployment_infrastructure "github.com/alex-galey/dokku-mcp/internal/server-plugins/deployment/infrastructure"
	"github.com/alex-galey/dokku-mcp/internal/shared"
	"go.uber.org/fx"
)

var Module = fx.Module("deployment",
	fx.Provide(
		// Deployment repository
		fx.Annotate(
			func(logger *slog.Logger) domain.DeploymentRepository {
				return deployment_infrastructure.NewDeploymentRepository(logger)
			},
		),
		// Deployment tracker
		fx.Annotate(
			domain.NewDeploymentTracker,
		),
		// Deployment status checker
		fx.Annotate(
			func(client dokkuApi.DokkuClient) domain.DeploymentStatusChecker {
				return deployment_infrastructure.NewDeploymentStatusChecker(client)
			},
		),
		// Deployment poller
		fx.Annotate(
			func(
				tracker *domain.DeploymentTracker,
				statusChecker domain.DeploymentStatusChecker,
				logger *slog.Logger,
			) *domain.DeploymentPoller {
				return domain.NewDeploymentPoller(
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
			domain.NewApplicationDeploymentService,
			fx.As(new(domain.DeploymentService)),
		),
		// Deployment adapter
		fx.Annotate(
			adapter.NewDeploymentServiceAdapter,
			fx.As(new(shared.DeploymentService)),
		),
	),
)

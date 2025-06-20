package deployment

import (
	"log/slog"

	dokkuApi "github.com/alex-galey/dokku-mcp/internal/dokku-api"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/deployment/adapter"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/deployment/domain"
	deployment_infrastructure "github.com/alex-galey/dokku-mcp/internal/server-plugins/deployment/infrastructure"
	"github.com/alex-galey/dokku-mcp/internal/shared"
	"go.uber.org/fx"
)

var Module = fx.Module("deployment",
	fx.Provide(
		fx.Annotate(
			func(client dokkuApi.DokkuClient, logger *slog.Logger) domain.DeploymentInfrastructure {
				return deployment_infrastructure.NewDeploymentInfrastructure(client, logger)
			},
		),
		fx.Annotate(
			func(logger *slog.Logger) domain.DeploymentRepository {
				return deployment_infrastructure.NewDeploymentRepository(logger)
			},
		),
		fx.Annotate(
			domain.NewApplicationDeploymentService,
			fx.As(new(domain.DeploymentService)),
		),
		fx.Annotate(
			adapter.NewDeploymentServiceAdapter,
			fx.As(new(shared.DeploymentService)),
		),
	),
)

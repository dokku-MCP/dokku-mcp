package domain

import (
	deployment_services "github.com/alex-galey/dokku-mcp/internal/domain/dokku/deployment/services"
	"go.uber.org/fx"
)

// DomainProviders provides all domain layer dependencies
var DomainProviders = fx.Options(
	fx.Provide(
		fx.Annotate(
			deployment_services.NewApplicationDeploymentService,
			fx.As(new(deployment_services.DeploymentService)),
		),
	),
)

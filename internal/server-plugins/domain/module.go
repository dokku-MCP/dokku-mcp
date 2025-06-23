package domain

import (
	serverDomain "github.com/alex-galey/dokku-mcp/internal/server-plugin/domain"
	"go.uber.org/fx"
)

var Module = fx.Module("domain",
	fx.Provide(
		fx.Annotate(
			NewDomainServerPlugin,
			fx.As(new(serverDomain.ServerPlugin)),
			fx.ResultTags(`group:"server_plugins"`),
		),
	),
)

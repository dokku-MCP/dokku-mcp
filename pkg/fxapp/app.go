package fxapp

import (
	"github.com/alex-galey/dokku-mcp/internal/server"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/app"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/core"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/deployment"
	"github.com/alex-galey/dokku-mcp/pkg/config"
	"github.com/alex-galey/dokku-mcp/pkg/logger"
	"go.uber.org/fx"
)

func New() *fx.App {
	return fx.New(
		config.Module,
		logger.Module,
		server.Module,
		core.CoreModule,
		deployment.Module,
		app.Module,
	)
}

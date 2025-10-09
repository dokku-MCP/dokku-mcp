package fxapp

import (
	"log"

	"github.com/alex-galey/dokku-mcp/internal/server"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/app"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/core"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/deployment"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/domain"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/onboarding"
	"github.com/alex-galey/dokku-mcp/pkg/config"
	"github.com/alex-galey/dokku-mcp/pkg/logger"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

func New() *fx.App {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	// Default to a verbose logger for debug level
	var fxLogger fx.Option = fx.WithLogger(
		func() fxevent.Logger {
			return &fxevent.ConsoleLogger{W: log.Writer()}
		},
	)

	if cfg.LogLevel != "debug" {
		fxLogger = fx.NopLogger
	}

	return fx.New(
		fxLogger,
		fx.Supply(cfg),
		config.Module,
		logger.Module,
		server.Module,
		core.CoreModule,
		domain.Module,
		deployment.Module,
		onboarding.Module,
		app.Module,
	)
}

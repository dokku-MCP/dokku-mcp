package dokkutesting

import (
	"context"
	"log/slog"
	"time"

	"github.com/alex-galey/dokku-mcp/src/domain/application"
)

// DokkuClient interface for Dokku operations (avoids import cycle)
type DokkuClient interface {
	ExecuteCommand(ctx context.Context, command string, args []string) ([]byte, error)
	GetApplications(ctx context.Context) ([]string, error)
	GetApplicationInfo(ctx context.Context, appName string) (map[string]interface{}, error)
	GetApplicationConfig(ctx context.Context, appName string) (map[string]string, error)
	SetApplicationConfig(ctx context.Context, appName string, config map[string]string) error
}

// ClientConfig configuration for creating a Dokku client
type ClientConfig struct {
	DokkuHost       string
	DokkuPort       int
	DokkuUser       string
	SSHKeyPath      string
	CommandTimeout  time.Duration
	AllowedCommands map[string]bool
}

// DokkuClientFactory interface for creating Dokku clients
type DokkuClientFactory interface {
	CreateClient(config *ClientConfig, logger *slog.Logger) DokkuClient
}

// DeploymentServiceFactory interface for creating deployment services
type DeploymentServiceFactory interface {
	CreateService(client DokkuClient, logger *slog.Logger) application.DeploymentService
}

package dokku

import (
	"log/slog"

	"github.com/alex-galey/dokku-mcp/internal/infrastructure/config"
)

// NewDokkuClientFromConfig creates a DokkuClient from the server configuration.
func NewDokkuClientFromConfig(cfg *config.ServerConfig, logger *slog.Logger) DokkuClient {
	dokkuConfig := &ClientConfig{
		DokkuHost:      cfg.DokkuHost,
		DokkuPort:      cfg.DokkuPort,
		DokkuUser:      cfg.DokkuUser,
		DokkuPath:      cfg.DokkuPath,
		SSHKeyPath:     cfg.SSHKeyPath,
		CommandTimeout: cfg.Timeout,
	}

	return NewDokkuClient(dokkuConfig, logger)
}

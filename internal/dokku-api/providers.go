package dokkuApi

import (
	"log/slog"

	"github.com/alex-galey/dokku-mcp/pkg/config"
)

// NewDokkuClientFromConfig creates a DokkuClient from the server configuration.
func NewDokkuClientFromConfig(cfg *config.ServerConfig, logger *slog.Logger) DokkuClient {
	sshHost := cfg.SSH.Host
	sshPort := cfg.SSH.Port
	sshUser := cfg.SSH.User
	sshKeyPath := cfg.SSH.KeyPath

	dokkuConfig := &ClientConfig{
		DokkuHost:      sshHost,
		DokkuPort:      sshPort,
		DokkuUser:      sshUser,
		DokkuPath:      cfg.DokkuPath,
		SSHKeyPath:     sshKeyPath,
		CommandTimeout: cfg.Timeout,
		Cache:          createCacheConfig(cfg),
	}

	client := NewDokkuClient(dokkuConfig, logger)
	client.SetBlacklist(cfg.Security.Blacklist)

	if cfg.CacheEnabled {
		logger.Info("Command-level caching enabled",
			"cache_ttl", cfg.CacheTTL)
	} else {
		logger.Info("Caching disabled")
	}

	return client
}

// createCacheConfig creates a cache configuration from server config
func createCacheConfig(cfg *config.ServerConfig) *CacheConfig {
	if !cfg.CacheEnabled {
		return &CacheConfig{
			Enabled: false,
		}
	}

	cacheConfig := DefaultCacheConfig()
	cacheConfig.Enabled = true

	// Use server config TTL if provided, otherwise use default
	if cfg.CacheTTL > 0 {
		cacheConfig.DefaultTTL = cfg.CacheTTL
	}

	return cacheConfig
}

// NewSSHConnectionManagerFromConfig creates an SSH connection manager directly from server configuration
func NewSSHConnectionManagerFromConfig(cfg *config.ServerConfig, logger *slog.Logger) (*SSHConnectionManager, error) {
	return NewSSHConnectionManagerFromServerConfig(cfg, logger)
}

package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type TransportConfig struct {
	Type string `mapstructure:"type"` // "stdio" or "sse"
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type SSHConfig struct {
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
	User    string `mapstructure:"user"`
	KeyPath string `mapstructure:"key_path"`
}

type PluginDiscoveryConfig struct {
	SyncInterval time.Duration `mapstructure:"sync_interval"`
	Enabled      bool          `mapstructure:"enabled"`
}

type SecurityConfig struct {
	Blacklist []string `mapstructure:"blacklist"`
}

type ServerConfig struct {
	Transport       TransportConfig       `mapstructure:"transport"`
	Host            string                `mapstructure:"host"`
	Port            int                   `mapstructure:"port"`
	LogLevel        string                `mapstructure:"log_level"`
	LogFormat       string                `mapstructure:"log_format"`
	Timeout         time.Duration         `mapstructure:"timeout"`
	DokkuPath       string                `mapstructure:"dokku_path"`
	CacheEnabled    bool                  `mapstructure:"cache_enabled"`
	CacheTTL        time.Duration         `mapstructure:"cache_ttl"`
	SSH             SSHConfig             `mapstructure:"ssh"`
	PluginDiscovery PluginDiscoveryConfig `mapstructure:"plugin_discovery"`
	Security        SecurityConfig        `mapstructure:"security"`
}

func DefaultConfig() *ServerConfig {
	return &ServerConfig{
		Transport: TransportConfig{
			Type: "stdio",
			Host: "localhost",
			Port: 8080,
		},
		Host:         "localhost",
		Port:         8080,
		LogLevel:     "info",
		LogFormat:    "json",
		Timeout:      30 * time.Second,
		DokkuPath:    "/usr/bin/dokku",
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
		SSH: SSHConfig{
			Host:    "localhost",
			Port:    3022,
			User:    "dokku",
			KeyPath: "dokku_mcp_test",
		},
		PluginDiscovery: PluginDiscoveryConfig{
			SyncInterval: 1 * time.Minute,
			Enabled:      true,
		},
		Security: SecurityConfig{
			Blacklist: []string{},
		},
	}
}

func LoadConfig() (*ServerConfig, error) {
	config := DefaultConfig()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/dokku-mcp/")
	viper.AddConfigPath("$HOME/.dokku-mcp/")

	viper.SetEnvPrefix("DOKKU_MCP")
	viper.AutomaticEnv()

	// Server configuration defaults
	viper.SetDefault("transport.type", config.Transport.Type)
	viper.SetDefault("transport.host", config.Transport.Host)
	viper.SetDefault("transport.port", config.Transport.Port)
	viper.SetDefault("host", config.Host)
	viper.SetDefault("port", config.Port)
	viper.SetDefault("log_level", config.LogLevel)
	viper.SetDefault("log_format", config.LogFormat)
	viper.SetDefault("timeout", config.Timeout)
	viper.SetDefault("dokku_path", config.DokkuPath)
	viper.SetDefault("cache_enabled", config.CacheEnabled)
	viper.SetDefault("cache_ttl", config.CacheTTL)

	// SSH configuration defaults
	viper.SetDefault("ssh.host", config.SSH.Host)
	viper.SetDefault("ssh.port", config.SSH.Port)
	viper.SetDefault("ssh.user", config.SSH.User)
	viper.SetDefault("ssh.key_path", config.SSH.KeyPath)

	// Plugin discovery configuration defaults
	viper.SetDefault("plugin_discovery.sync_interval", config.PluginDiscovery.SyncInterval)
	viper.SetDefault("plugin_discovery.enabled", config.PluginDiscovery.Enabled)

	// Security configuration defaults
	viper.SetDefault("security.blacklist", config.Security.Blacklist)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read configuration file: %w", err)
		}
	}

	// Decode the configuration
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

func validateConfig(config *ServerConfig) error {
	if config.Port <= 0 || config.Port > 65535 {
		return fmt.Errorf("the port must be between 1 and 65535")
	}

	if config.Timeout <= 0 {
		return fmt.Errorf("the timeout must be positive")
	}

	if config.DokkuPath == "" {
		return fmt.Errorf("the Dokku path cannot be empty")
	}

	// Validate SSH configuration
	if config.SSH.Host == "" {
		return fmt.Errorf("the SSH host cannot be empty")
	}

	if config.SSH.Port <= 0 || config.SSH.Port > 65535 {
		return fmt.Errorf("the SSH port must be between 1 and 65535")
	}

	if config.SSH.User == "" {
		return fmt.Errorf("the SSH user cannot be empty")
	}

	validLogLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "error": true,
	}
	if !validLogLevels[config.LogLevel] {
		return fmt.Errorf("invalid log level: %s", config.LogLevel)
	}

	validLogFormats := map[string]bool{
		"json": true, "text": true,
	}
	if !validLogFormats[config.LogFormat] {
		return fmt.Errorf("invalid log format: %s", config.LogFormat)
	}

	return nil
}

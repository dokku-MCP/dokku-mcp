package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

//go:generate go run ../../cmd/gen-mcp-json

type TransportConfig struct {
	Type string     `mapstructure:"type"` // "stdio" or "sse"
	Host string     `mapstructure:"host"`
	Port int        `mapstructure:"port"`
	CORS CORSConfig `mapstructure:"cors"`
}

type CORSConfig struct {
	Enabled        bool     `mapstructure:"enabled"`
	AllowedOrigins []string `mapstructure:"allowed_origins"` // If empty and enabled, uses "*"
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
	MaxAge         int      `mapstructure:"max_age"` // In seconds
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

type MultiTenantConfig struct {
	Enabled        bool                 `mapstructure:"enabled"`
	Authentication AuthenticationConfig `mapstructure:"authentication"`
	Authorization  AuthorizationConfig  `mapstructure:"authorization"`
	Observability  ObservabilityConfig  `mapstructure:"observability"`
}

type AuthenticationConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	JWTSecret       string `mapstructure:"jwt_secret"`
	TokenHeader     string `mapstructure:"token_header"`
	TokenQueryParam string `mapstructure:"token_query_param"`
}

type AuthorizationConfig struct {
	Enabled            bool     `mapstructure:"enabled"`
	DefaultPermissions []string `mapstructure:"default_permissions"`
}

type ObservabilityConfig struct {
	AuditEnabled   bool `mapstructure:"audit_enabled"`
	MetricsEnabled bool `mapstructure:"metrics_enabled"`
	TracingEnabled bool `mapstructure:"tracing_enabled"`
}

type LogsConfig struct {
	Runtime RuntimeLogsConfig `mapstructure:"runtime"`
	Build   BuildLogsConfig   `mapstructure:"build"`
}

type RuntimeLogsConfig struct {
	DefaultLines     int `mapstructure:"default_lines"`
	MaxLines         int `mapstructure:"max_lines"`
	StreamBufferSize int `mapstructure:"stream_buffer_size"`
}

type BuildLogsConfig struct {
	MaxSizeMB int           `mapstructure:"max_size_mb"`
	Retention time.Duration `mapstructure:"retention"`
}

type ServerConfig struct {
	Transport          TransportConfig       `mapstructure:"transport"`
	Host               string                `mapstructure:"host"`
	Port               int                   `mapstructure:"port"`
	LogLevel           string                `mapstructure:"log_level"`
	LogFormat          string                `mapstructure:"log_format"`
	ExposeServerLogs   bool                  `mapstructure:"expose_server_logs"`
	LogBufferCapacity  int                   `mapstructure:"log_buffer_capacity"`
	DeploymentLogLines int                   `mapstructure:"deployment_log_lines"`
	Timeout            time.Duration         `mapstructure:"timeout"`
	DokkuPath          string                `mapstructure:"dokku_path"`
	CacheEnabled       bool                  `mapstructure:"cache_enabled"`
	CacheTTL           time.Duration         `mapstructure:"cache_ttl"`
	SSH                SSHConfig             `mapstructure:"ssh"`
	PluginDiscovery    PluginDiscoveryConfig `mapstructure:"plugin_discovery"`
	Security           SecurityConfig        `mapstructure:"security"`
	MultiTenant        MultiTenantConfig     `mapstructure:"multi_tenant"`
	Logs               LogsConfig            `mapstructure:"logs"`
}

func DefaultConfig() *ServerConfig {
	return &ServerConfig{
		Transport: TransportConfig{
			Type: "stdio",
			Host: "localhost",
			Port: 8080,
			CORS: CORSConfig{
				Enabled:        false, // Disabled by default, mcp-go handles CORS with "*"
				AllowedOrigins: []string{},
				AllowedMethods: []string{"GET", "POST", "OPTIONS"},
				AllowedHeaders: []string{"Content-Type", "Authorization"},
				MaxAge:         300, // 5 minutes
			},
		},
		Host:               "localhost",
		Port:               8080,
		LogLevel:           "info",
		LogFormat:          "json",
		ExposeServerLogs:   false,
		LogBufferCapacity:  2000,
		DeploymentLogLines: 200,
		Timeout:            30 * time.Second,
		DokkuPath:          "/usr/bin/dokku",
		CacheEnabled:       true,
		CacheTTL:           5 * time.Minute,
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
		MultiTenant: MultiTenantConfig{
			Enabled: false,
			Authentication: AuthenticationConfig{
				Enabled:         false,
				TokenHeader:     "Authorization",
				TokenQueryParam: "token",
			},
			Authorization: AuthorizationConfig{
				Enabled:            false,
				DefaultPermissions: []string{},
			},
			Observability: ObservabilityConfig{
				AuditEnabled:   false,
				MetricsEnabled: false,
				TracingEnabled: false,
			},
		},
		Logs: LogsConfig{
			Runtime: RuntimeLogsConfig{
				DefaultLines:     100,
				MaxLines:         1000,
				StreamBufferSize: 1000,
			},
			Build: BuildLogsConfig{
				MaxSizeMB: 10,
				Retention: 5 * time.Minute,
			},
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
	viper.SetDefault("transport.cors.enabled", config.Transport.CORS.Enabled)
	viper.SetDefault("transport.cors.allowed_origins", config.Transport.CORS.AllowedOrigins)
	viper.SetDefault("transport.cors.allowed_methods", config.Transport.CORS.AllowedMethods)
	viper.SetDefault("transport.cors.allowed_headers", config.Transport.CORS.AllowedHeaders)
	viper.SetDefault("transport.cors.max_age", config.Transport.CORS.MaxAge)
	viper.SetDefault("host", config.Host)
	viper.SetDefault("port", config.Port)
	viper.SetDefault("log_level", config.LogLevel)
	viper.SetDefault("log_format", config.LogFormat)
	viper.SetDefault("expose_server_logs", config.ExposeServerLogs)
	viper.SetDefault("log_buffer_capacity", config.LogBufferCapacity)
	viper.SetDefault("deployment_log_lines", config.DeploymentLogLines)
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

	// Logs configuration defaults
	viper.SetDefault("logs.runtime.default_lines", config.Logs.Runtime.DefaultLines)
	viper.SetDefault("logs.runtime.max_lines", config.Logs.Runtime.MaxLines)
	viper.SetDefault("logs.runtime.stream_buffer_size", config.Logs.Runtime.StreamBufferSize)
	viper.SetDefault("logs.build.max_size_mb", config.Logs.Build.MaxSizeMB)
	viper.SetDefault("logs.build.retention", config.Logs.Build.Retention)

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

	// Validate logs configuration
	if config.Logs.Runtime.DefaultLines <= 0 || config.Logs.Runtime.DefaultLines > 100000 {
		return fmt.Errorf("logs.runtime.default_lines must be between 1 and 100000")
	}
	if config.Logs.Runtime.MaxLines <= 0 || config.Logs.Runtime.MaxLines > 100000 {
		return fmt.Errorf("logs.runtime.max_lines must be between 1 and 100000")
	}
	if config.Logs.Runtime.StreamBufferSize <= 0 || config.Logs.Runtime.StreamBufferSize > 10000 {
		return fmt.Errorf("logs.runtime.stream_buffer_size must be between 1 and 10000")
	}
	if config.Logs.Build.MaxSizeMB <= 0 || config.Logs.Build.MaxSizeMB > 100 {
		return fmt.Errorf("logs.build.max_size_mb must be between 1 and 100")
	}
	if config.Logs.Build.Retention <= 0 {
		return fmt.Errorf("logs.build.retention must be positive")
	}

	return nil
}

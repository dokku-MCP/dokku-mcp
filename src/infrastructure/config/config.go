package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type ServerConfig struct {
	Host            string          `mapstructure:"host"`
	Port            int             `mapstructure:"port"`
	LogLevel        string          `mapstructure:"log_level"`
	LogFormat       string          `mapstructure:"log_format"`
	Timeout         time.Duration   `mapstructure:"timeout"`
	DokkuPath       string          `mapstructure:"dokku_path"`
	CacheEnabled    bool            `mapstructure:"cache_enabled"`
	CacheTTL        time.Duration   `mapstructure:"cache_ttl"`
	DokkuHost       string          `mapstructure:"dokku_host"`
	DokkuPort       int             `mapstructure:"dokku_port"`
	DokkuUser       string          `mapstructure:"dokku_user"`
	SSHKeyPath      string          `mapstructure:"ssh_key_path"`
	AllowedCommands map[string]bool `mapstructure:"security.allowed_commands"`
}

func DefaultConfig() *ServerConfig {
	return &ServerConfig{
		Host:         "localhost",
		Port:         8080,
		LogLevel:     "info",
		LogFormat:    "json",
		Timeout:      30 * time.Second,
		DokkuPath:    "/usr/bin/dokku",
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
		DokkuHost:    "dokku.me",
		DokkuPort:    22,
		DokkuUser:    "user",
		SSHKeyPath:   "",
		AllowedCommands: map[string]bool{
			"apps:list":    true,
			"apps:info":    true,
			"apps:create":  true,
			"apps:exists":  true,
			"config:get":   true,
			"config:set":   true,
			"config:show":  true,
			"domains:add":  true,
			"domains:list": true,
			"ps:scale":     true,
			"logs":         true,
			"events":       true,
			"git:report":   true,
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

	viper.SetDefault("host", config.Host)
	viper.SetDefault("port", config.Port)
	viper.SetDefault("log_level", config.LogLevel)
	viper.SetDefault("log_format", config.LogFormat)
	viper.SetDefault("timeout", config.Timeout)
	viper.SetDefault("dokku_path", config.DokkuPath)
	viper.SetDefault("cache_enabled", config.CacheEnabled)
	viper.SetDefault("cache_ttl", config.CacheTTL)
	viper.SetDefault("dokku_host", config.DokkuHost)
	viper.SetDefault("dokku_port", config.DokkuPort)
	viper.SetDefault("dokku_user", config.DokkuUser)
	viper.SetDefault("ssh_key_path", config.SSHKeyPath)

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

	if config.DokkuHost == "" {
		return fmt.Errorf("the Dokku host cannot be empty")
	}

	if config.DokkuPort <= 0 || config.DokkuPort > 65535 {
		return fmt.Errorf("the SSH port must be between 1 and 65535")
	}

	if config.DokkuUser == "" {
		return fmt.Errorf("the Dokku user cannot be empty")
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

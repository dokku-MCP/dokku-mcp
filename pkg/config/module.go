package config

import "go.uber.org/fx"

var Module = fx.Module("config",
	fx.Provide(LoadConfig), // Provides the full ServerConfig
	// Provides specific, smaller configs for consumers
	fx.Provide(func(cfg *ServerConfig) TransportConfig { return cfg.Transport }),
	fx.Provide(func(cfg *ServerConfig) SSHConfig { return cfg.SSH }),
)

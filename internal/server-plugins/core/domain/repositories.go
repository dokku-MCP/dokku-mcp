package domain

import (
	"context"
)

// SystemRepository defines methods for accessing system-level information
type SystemRepository interface {
	GetSystemStatus(ctx context.Context) (*SystemStatus, error)
	GetServerInfo(ctx context.Context) (*ServerInfo, error)
	GetResourceUsage(ctx context.Context) (*ResourceUsage, error)
}

// PluginRepository defines methods for managing Dokku plugins
type PluginRepository interface {
	ListPlugins(ctx context.Context) ([]DokkuPlugin, error)
	GetPlugin(ctx context.Context, name string) (*DokkuPlugin, error)
	InstallPlugin(ctx context.Context, source string, options map[string]string) error
	UninstallPlugin(ctx context.Context, name string) error
	EnablePlugin(ctx context.Context, name string) error
	DisablePlugin(ctx context.Context, name string) error
	UpdatePlugin(ctx context.Context, name string, version string) error
}

// SSHKeyRepository defines methods for managing SSH keys
type SSHKeyRepository interface {
	ListSSHKeys(ctx context.Context) ([]SSHKey, error)
	AddSSHKey(ctx context.Context, name string, keyContent string) error
	RemoveSSHKey(ctx context.Context, name string) error
	GetSSHKey(ctx context.Context, name string) (*SSHKey, error)
}

// RegistryRepository defines methods for managing Docker registry credentials
type RegistryRepository interface {
	ListRegistries(ctx context.Context) ([]RegistryCredential, error)
	// LoginRegistry stores registry credentials for one app, or globally
	// when appName is empty.
	LoginRegistry(ctx context.Context, appName, registry, username, password string) error
	// LogoutRegistry removes registry credentials for one app, or globally
	// when appName is empty.
	LogoutRegistry(ctx context.Context, appName, registry string) error
	GetRegistryStatus(ctx context.Context, registry string) (*RegistryCredential, error)
	// GetRegistryReport returns registry settings for one app, or the global
	// settings when appName is empty.
	GetRegistryReport(ctx context.Context, appName string) (map[string]string, error)
}

// ConfigurationRepository defines methods for managing global configuration
type ConfigurationRepository interface {
	GetGlobalConfiguration(ctx context.Context) (*GlobalConfiguration, error)
	SetGlobalProxyType(ctx context.Context, proxyType string) error
	SetGlobalScheduler(ctx context.Context, scheduler string) error
	SetGlobalDeployBranch(ctx context.Context, branch string) error
	SetVectorSink(ctx context.Context, sink string) error
	GetConfigurationKeys(ctx context.Context, scope string) ([]ConfigurationKey, error)
}

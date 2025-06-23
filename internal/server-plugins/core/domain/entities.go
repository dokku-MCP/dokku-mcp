package domain

import (
	"time"
)

// SystemStatus represents the overall status of the Dokku server
type SystemStatus struct {
	Version       string            `json:"version"`
	Status        string            `json:"status"`
	Hostname      string            `json:"hostname"`
	GlobalDomains []string          `json:"global_domains"`
	ProxyType     string            `json:"proxy_type"`
	Scheduler     string            `json:"scheduler"`
	InstalledAt   time.Time         `json:"installed_at"`
	LastUpdated   time.Time         `json:"last_updated"`
	Configuration map[string]string `json:"configuration"`
}

// DokkuPlugin represents a Dokku plugin
type DokkuPlugin struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Status      string `json:"status"` // enabled, disabled
	Description string `json:"description"`
	Source      string `json:"source"`
	CorePlugin  bool   `json:"core_plugin"`
}

// SSHKey represents an SSH key in Dokku
type SSHKey struct {
	Name        string    `json:"name"`
	Fingerprint string    `json:"fingerprint"`
	KeyType     string    `json:"key_type"`
	Comment     string    `json:"comment"`
	AddedAt     time.Time `json:"added_at"`
}

// RegistryCredential represents Docker registry credentials
type RegistryCredential struct {
	Registry string    `json:"registry"`
	Username string    `json:"username"`
	AddedAt  time.Time `json:"added_at"`
	Active   bool      `json:"active"`
}

// GlobalConfiguration represents global Dokku configuration
type GlobalConfiguration struct {
	ProxyType     string            `json:"proxy_type"`
	Scheduler     string            `json:"scheduler"`
	DeployBranch  string            `json:"deploy_branch"`
	LetsEncrypt   LetsEncryptConfig `json:"letsencrypt"`
	VectorSink    string            `json:"vector_sink"`
	StorageVolume string            `json:"storage_volume"`
	CustomVars    map[string]string `json:"custom_vars"`
}

// ServerInfo represents comprehensive server information
type ServerInfo struct {
	SystemStatus  SystemStatus         `json:"system_status"`
	Plugins       []DokkuPlugin        `json:"plugins"`
	SSHKeys       []SSHKey             `json:"ssh_keys"`
	Registries    []RegistryCredential `json:"registries"`
	Configuration GlobalConfiguration  `json:"configuration"`
	ResourceUsage ResourceUsage        `json:"resource_usage"`
}

// ResourceUsage represents system resource usage
type ResourceUsage struct {
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	DiskUsage   float64   `json:"disk_usage"`
	NetworkIn   float64   `json:"network_in"`
	NetworkOut  float64   `json:"network_out"`
	LastUpdated time.Time `json:"last_updated"`
}

// ConfigurationKey represents a configuration key-value pair
type ConfigurationKey struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	Scope       string    `json:"scope"` // global, app-specific
	SetAt       time.Time `json:"set_at"`
}

// LetsEncryptConfig represents Let's Encrypt configuration
type LetsEncryptConfig struct {
	Email           string `json:"email"`
	Server          string `json:"server,omitempty"`
	Enabled         bool   `json:"enabled"`
	AutoRenewHour   int    `json:"auto_renew_hour,omitempty"`
	GracePeriodDays int    `json:"grace_period_days,omitempty"`
}

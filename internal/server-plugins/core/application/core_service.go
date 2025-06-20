package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/alex-galey/dokku-mcp/internal/server-plugins/core/domain"
)

// CoreService provides application-level orchestration for core Dokku functionality
type CoreService struct {
	systemRepo   domain.SystemRepository
	pluginRepo   domain.PluginRepository
	domainRepo   domain.DomainRepository
	sshKeyRepo   domain.SSHKeyRepository
	registryRepo domain.RegistryRepository
	configRepo   domain.ConfigurationRepository
	logger       *slog.Logger
}

// NewCoreService creates a new core application service
func NewCoreService(
	systemRepo domain.SystemRepository,
	pluginRepo domain.PluginRepository,
	domainRepo domain.DomainRepository,
	sshKeyRepo domain.SSHKeyRepository,
	registryRepo domain.RegistryRepository,
	configRepo domain.ConfigurationRepository,
	logger *slog.Logger,
) *CoreService {
	return &CoreService{
		systemRepo:   systemRepo,
		pluginRepo:   pluginRepo,
		domainRepo:   domainRepo,
		sshKeyRepo:   sshKeyRepo,
		registryRepo: registryRepo,
		configRepo:   configRepo,
		logger:       logger,
	}
}

// System Status Operations
func (s *CoreService) GetSystemStatus(ctx context.Context) (*domain.SystemStatus, error) {
	s.logger.Debug("Getting system status")
	return s.systemRepo.GetSystemStatus(ctx)
}

func (s *CoreService) GetServerInfo(ctx context.Context) (*domain.ServerInfo, error) {
	s.logger.Debug("Getting complete server information")
	return s.systemRepo.GetServerInfo(ctx)
}

func (s *CoreService) GetResourceUsage(ctx context.Context) (*domain.ResourceUsage, error) {
	s.logger.Debug("Getting resource usage")
	return s.systemRepo.GetResourceUsage(ctx)
}

// Plugin Management Operations
func (s *CoreService) ListPlugins(ctx context.Context) ([]domain.DokkuPlugin, error) {
	s.logger.Debug("Listing Dokku plugins")
	return s.pluginRepo.ListPlugins(ctx)
}

func (s *CoreService) GetPlugin(ctx context.Context, name string) (*domain.DokkuPlugin, error) {
	s.logger.Debug("Getting plugin information", "plugin", name)

	if name == "" {
		return nil, fmt.Errorf("plugin name cannot be empty")
	}

	return s.pluginRepo.GetPlugin(ctx, name)
}

func (s *CoreService) InstallPlugin(ctx context.Context, source string, options map[string]string) error {
	s.logger.Info("Installing plugin", "source", source, "options", options)

	if source == "" {
		return fmt.Errorf("plugin source cannot be empty")
	}

	// Validate source URL format if needed
	if err := s.validatePluginSource(source); err != nil {
		return fmt.Errorf("invalid plugin source: %w", err)
	}

	return s.pluginRepo.InstallPlugin(ctx, source, options)
}

func (s *CoreService) UninstallPlugin(ctx context.Context, name string) error {
	s.logger.Info("Uninstalling plugin", "plugin", name)

	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	// Check if plugin exists before trying to uninstall
	_, err := s.pluginRepo.GetPlugin(ctx, name)
	if err != nil {
		return fmt.Errorf("plugin not found: %w", err)
	}

	return s.pluginRepo.UninstallPlugin(ctx, name)
}

func (s *CoreService) TogglePlugin(ctx context.Context, name string, enable bool) error {
	action := "disable"
	if enable {
		action = "enable"
	}

	s.logger.Info("Toggling plugin", "plugin", name, "action", action)

	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	if enable {
		return s.pluginRepo.EnablePlugin(ctx, name)
	}
	return s.pluginRepo.DisablePlugin(ctx, name)
}

func (s *CoreService) UpdatePlugin(ctx context.Context, name string, version string) error {
	s.logger.Info("Updating plugin", "plugin", name, "version", version)

	if name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	return s.pluginRepo.UpdatePlugin(ctx, name, version)
}

// Domain Management Operations
func (s *CoreService) ListGlobalDomains(ctx context.Context) ([]domain.GlobalDomain, error) {
	s.logger.Debug("Listing global domains")
	return s.domainRepo.ListGlobalDomains(ctx)
}

func (s *CoreService) AddGlobalDomain(ctx context.Context, domainName string) error {
	s.logger.Info("Adding global domain", "domain", domainName)

	if err := s.validateDomainName(domainName); err != nil {
		return fmt.Errorf("invalid domain name: %w", err)
	}

	return s.domainRepo.AddGlobalDomain(ctx, domainName)
}

func (s *CoreService) RemoveGlobalDomain(ctx context.Context, domainName string) error {
	s.logger.Info("Removing global domain", "domain", domainName)

	if domainName == "" {
		return fmt.Errorf("domain name cannot be empty")
	}

	return s.domainRepo.RemoveGlobalDomain(ctx, domainName)
}

func (s *CoreService) SetGlobalDomains(ctx context.Context, domains []string) error {
	s.logger.Info("Setting global domains", "domains", domains)

	// Validate all domain names
	for _, domainName := range domains {
		if err := s.validateDomainName(domainName); err != nil {
			return fmt.Errorf("invalid domain name '%s': %w", domainName, err)
		}
	}

	return s.domainRepo.SetGlobalDomains(ctx, domains)
}

func (s *CoreService) ClearGlobalDomains(ctx context.Context) error {
	s.logger.Info("Clearing all global domains")
	return s.domainRepo.ClearGlobalDomains(ctx)
}

func (s *CoreService) GetDomainsReport(ctx context.Context) (*domain.DomainsReport, error) {
	s.logger.Debug("Getting domains report")
	return s.domainRepo.GetDomainsReport(ctx)
}

// SSH Key Management Operations
func (s *CoreService) ListSSHKeys(ctx context.Context) ([]domain.SSHKey, error) {
	s.logger.Debug("Listing SSH keys")
	return s.sshKeyRepo.ListSSHKeys(ctx)
}

func (s *CoreService) AddSSHKey(ctx context.Context, name string, keyContent string) error {
	s.logger.Info("Adding SSH key", "key_name", name)

	if name == "" {
		return fmt.Errorf("SSH key name cannot be empty")
	}

	if keyContent == "" {
		return fmt.Errorf("SSH key content cannot be empty")
	}

	// Basic validation of SSH key format
	if err := s.validateSSHKeyContent(keyContent); err != nil {
		return fmt.Errorf("invalid SSH key format: %w", err)
	}

	return s.sshKeyRepo.AddSSHKey(ctx, name, keyContent)
}

func (s *CoreService) RemoveSSHKey(ctx context.Context, name string) error {
	s.logger.Info("Removing SSH key", "key_name", name)

	if name == "" {
		return fmt.Errorf("SSH key name cannot be empty")
	}

	return s.sshKeyRepo.RemoveSSHKey(ctx, name)
}

// Registry Management Operations
func (s *CoreService) ListRegistries(ctx context.Context) ([]domain.RegistryCredential, error) {
	s.logger.Debug("Listing registries")
	return s.registryRepo.ListRegistries(ctx)
}

func (s *CoreService) LoginRegistry(ctx context.Context, registry, username, password string) error {
	s.logger.Info("Logging into registry", "registry", registry, "username", username)

	if registry == "" {
		return fmt.Errorf("registry URL cannot be empty")
	}

	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}

	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	return s.registryRepo.LoginRegistry(ctx, registry, username, password)
}

// Configuration Management Operations
func (s *CoreService) GetGlobalConfiguration(ctx context.Context) (*domain.GlobalConfiguration, error) {
	s.logger.Debug("Getting global configuration")
	return s.configRepo.GetGlobalConfiguration(ctx)
}

func (s *CoreService) SetGlobalProxyType(ctx context.Context, proxyType string) error {
	s.logger.Info("Setting global proxy type", "proxy_type", proxyType)

	validProxyTypes := []string{"nginx", "caddy", "traefik"}
	if !s.isValidProxyType(proxyType, validProxyTypes) {
		return fmt.Errorf("invalid proxy type '%s', must be one of: %v", proxyType, validProxyTypes)
	}

	return s.configRepo.SetGlobalProxyType(ctx, proxyType)
}

func (s *CoreService) SetGlobalScheduler(ctx context.Context, scheduler string) error {
	s.logger.Info("Setting global scheduler", "scheduler", scheduler)

	validSchedulers := []string{"docker-local", "k3s", "nomad"}
	if !s.isValidScheduler(scheduler, validSchedulers) {
		return fmt.Errorf("invalid scheduler '%s', must be one of: %v", scheduler, validSchedulers)
	}

	return s.configRepo.SetGlobalScheduler(ctx, scheduler)
}

func (s *CoreService) SetGlobalDeployBranch(ctx context.Context, branch string) error {
	s.logger.Info("Setting global deploy branch", "branch", branch)

	if branch == "" {
		return fmt.Errorf("deploy branch cannot be empty")
	}

	return s.configRepo.SetGlobalDeployBranch(ctx, branch)
}

// Validation helpers
func (s *CoreService) validatePluginSource(source string) error {
	// Basic validation - could be enhanced with more robust URL validation
	if len(source) < 5 {
		return fmt.Errorf("source too short")
	}

	// Check for common plugin source patterns
	if !(containsAny(source, []string{"github.com", "gitlab.com", "bitbucket.org", "git://", "https://"})) {
		return fmt.Errorf("source does not appear to be a valid repository URL")
	}

	return nil
}

func (s *CoreService) validateDomainName(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain name cannot be empty")
	}

	if len(domain) > 253 {
		return fmt.Errorf("domain name too long (max 253 characters)")
	}

	// Basic domain validation - could be enhanced with regex
	if !containsAny(domain, []string{".", "-"}) && !isIPAddress(domain) {
		return fmt.Errorf("invalid domain format")
	}

	return nil
}

func (s *CoreService) validateSSHKeyContent(keyContent string) error {
	// Basic SSH key validation
	if !containsAny(keyContent, []string{"ssh-rsa", "ssh-ed25519", "ssh-dss", "ecdsa-sha2"}) {
		return fmt.Errorf("key does not appear to be a valid SSH public key")
	}

	return nil
}

func (s *CoreService) isValidProxyType(proxyType string, validTypes []string) bool {
	for _, valid := range validTypes {
		if proxyType == valid {
			return true
		}
	}
	return false
}

func (s *CoreService) isValidScheduler(scheduler string, validSchedulers []string) bool {
	for _, valid := range validSchedulers {
		if scheduler == valid {
			return true
		}
	}
	return false
}

// Utility functions
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if len(s) >= len(substr) && findSubstring(s, substr) {
			return true
		}
	}
	return false
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func isIPAddress(s string) bool {
	// Simple IP address check
	parts := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			parts++
		} else if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return parts == 3
}

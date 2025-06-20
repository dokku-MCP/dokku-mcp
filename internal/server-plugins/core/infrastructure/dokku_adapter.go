package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	dokkuApi "github.com/alex-galey/dokku-mcp/internal/dokku-api"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/core/domain"
)

// DokkuCoreAdapter implements core domain repositories using Dokku CLI
type DokkuCoreAdapter struct {
	client dokkuApi.DokkuClient
	logger *slog.Logger
}

// NewDokkuCoreAdapter creates a new core adapter
func NewDokkuCoreAdapter(client dokkuApi.DokkuClient, logger *slog.Logger) *DokkuCoreAdapter {
	return &DokkuCoreAdapter{
		client: client,
		logger: logger,
	}
}

// SystemRepository implementation
func (a *DokkuCoreAdapter) GetSystemStatus(ctx context.Context) (*domain.SystemStatus, error) {
	status := &domain.SystemStatus{
		LastUpdated: time.Now(),
	}

	// Get version
	versionOutput, err := a.client.ExecuteCommand(ctx, "version", []string{})
	if err != nil {
		a.logger.Warn("Failed to get Dokku version", "error", err)
		status.Version = "unknown"
	} else {
		status.Version = strings.TrimSpace(string(versionOutput))
	}

	// Get global domains
	domainsOutput, err := a.client.ExecuteCommand(ctx, "domains:report", []string{"--global"})
	if err != nil {
		a.logger.Warn("Failed to get global domains", "error", err)
	} else {
		domains := a.parseGlobalDomains(string(domainsOutput))
		status.GlobalDomains = domains
	}

	// Get proxy type
	proxyOutput, err := a.client.ExecuteCommand(ctx, "proxy:report", []string{"--global", "--proxy-type"})
	if err != nil {
		a.logger.Warn("Failed to get proxy type", "error", err)
		status.ProxyType = "nginx" // default
	} else {
		status.ProxyType = strings.TrimSpace(string(proxyOutput))
	}

	// Get scheduler
	schedulerOutput, err := a.client.ExecuteCommand(ctx, "scheduler:report", []string{"--global", "--scheduler-selected"})
	if err != nil {
		a.logger.Warn("Failed to get scheduler", "error", err)
		status.Scheduler = "docker-local" // default
	} else {
		status.Scheduler = strings.TrimSpace(string(schedulerOutput))
	}

	status.Status = "running"
	return status, nil
}

func (a *DokkuCoreAdapter) GetServerInfo(ctx context.Context) (*domain.ServerInfo, error) {
	systemStatus, err := a.GetSystemStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get system status: %w", err)
	}

	plugins, err := a.ListPlugins(ctx)
	if err != nil {
		a.logger.Warn("Failed to get plugins", "error", err)
		plugins = []domain.DokkuPlugin{}
	}

	globalDomains, err := a.ListGlobalDomains(ctx)
	if err != nil {
		a.logger.Warn("Failed to get global domains", "error", err)
		globalDomains = []domain.GlobalDomain{}
	}

	sshKeys, err := a.ListSSHKeys(ctx)
	if err != nil {
		a.logger.Warn("Failed to get SSH keys", "error", err)
		sshKeys = []domain.SSHKey{}
	}

	registries, err := a.ListRegistries(ctx)
	if err != nil {
		a.logger.Warn("Failed to get registries", "error", err)
		registries = []domain.RegistryCredential{}
	}

	config, err := a.GetGlobalConfiguration(ctx)
	if err != nil {
		a.logger.Warn("Failed to get global configuration", "error", err)
		config = &domain.GlobalConfiguration{}
	}

	resourceUsage, err := a.GetResourceUsage(ctx)
	if err != nil {
		a.logger.Warn("Failed to get resource usage", "error", err)
		resourceUsage = &domain.ResourceUsage{LastUpdated: time.Now()}
	}

	return &domain.ServerInfo{
		SystemStatus:  *systemStatus,
		Plugins:       plugins,
		GlobalDomains: globalDomains,
		SSHKeys:       sshKeys,
		Registries:    registries,
		Configuration: *config,
		ResourceUsage: *resourceUsage,
	}, nil
}

func (a *DokkuCoreAdapter) GetResourceUsage(ctx context.Context) (*domain.ResourceUsage, error) {
	// This would typically involve getting system metrics
	// For now, returning basic placeholder data
	return &domain.ResourceUsage{
		CPUUsage:    0.0,
		MemoryUsage: 0.0,
		DiskUsage:   0.0,
		NetworkIn:   0.0,
		NetworkOut:  0.0,
		LastUpdated: time.Now(),
	}, nil
}

// PluginRepository implementation
func (a *DokkuCoreAdapter) ListPlugins(ctx context.Context) ([]domain.DokkuPlugin, error) {
	output, err := a.client.ExecuteCommand(ctx, "plugin:list", []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to list plugins: %w", err)
	}

	return a.parsePluginList(string(output)), nil
}

func (a *DokkuCoreAdapter) GetPlugin(ctx context.Context, name string) (*domain.DokkuPlugin, error) {
	plugins, err := a.ListPlugins(ctx)
	if err != nil {
		return nil, err
	}

	for _, plugin := range plugins {
		if plugin.Name == name {
			return &plugin, nil
		}
	}

	return nil, fmt.Errorf("plugin %s not found", name)
}

func (a *DokkuCoreAdapter) InstallPlugin(ctx context.Context, source string, options map[string]string) error {
	args := []string{source}

	// Add options
	for key, value := range options {
		switch key {
		case "committish":
			args = append(args, "--committish", value)
		case "name":
			args = append(args, "--name", value)
		case "core":
			if value == "true" {
				args = append(args, "--core")
			}
		}
	}

	_, err := a.client.ExecuteCommand(ctx, "plugin:install", args)
	if err != nil {
		return fmt.Errorf("failed to install plugin %s: %w", source, err)
	}

	return nil
}

func (a *DokkuCoreAdapter) UninstallPlugin(ctx context.Context, name string) error {
	_, err := a.client.ExecuteCommand(ctx, "plugin:uninstall", []string{name})
	if err != nil {
		return fmt.Errorf("failed to uninstall plugin %s: %w", name, err)
	}
	return nil
}

func (a *DokkuCoreAdapter) EnablePlugin(ctx context.Context, name string) error {
	_, err := a.client.ExecuteCommand(ctx, "plugin:enable", []string{name})
	if err != nil {
		return fmt.Errorf("failed to enable plugin %s: %w", name, err)
	}
	return nil
}

func (a *DokkuCoreAdapter) DisablePlugin(ctx context.Context, name string) error {
	_, err := a.client.ExecuteCommand(ctx, "plugin:disable", []string{name})
	if err != nil {
		return fmt.Errorf("failed to disable plugin %s: %w", name, err)
	}
	return nil
}

func (a *DokkuCoreAdapter) UpdatePlugin(ctx context.Context, name string, version string) error {
	args := []string{name}
	if version != "" {
		args = append(args, version)
	}

	_, err := a.client.ExecuteCommand(ctx, "plugin:update", args)
	if err != nil {
		return fmt.Errorf("failed to update plugin %s: %w", name, err)
	}
	return nil
}

// DomainRepository implementation
func (a *DokkuCoreAdapter) ListGlobalDomains(ctx context.Context) ([]domain.GlobalDomain, error) {
	output, err := a.client.ExecuteCommand(ctx, "domains:report", []string{"--global"})
	if err != nil {
		return nil, fmt.Errorf("failed to list global domains: %w", err)
	}

	domainStrings := a.parseGlobalDomains(string(output))
	var domains []domain.GlobalDomain

	for _, domainStr := range domainStrings {
		domains = append(domains, domain.GlobalDomain{
			Domain:     domainStr,
			IsWildcard: strings.HasPrefix(domainStr, "*."),
			AddedAt:    time.Now(), // We can't get the actual added time from Dokku
		})
	}

	return domains, nil
}

func (a *DokkuCoreAdapter) AddGlobalDomain(ctx context.Context, domainName string) error {
	_, err := a.client.ExecuteCommand(ctx, "domains:add-global", []string{domainName})
	if err != nil {
		return fmt.Errorf("failed to add global domain %s: %w", domainName, err)
	}
	return nil
}

func (a *DokkuCoreAdapter) RemoveGlobalDomain(ctx context.Context, domainName string) error {
	_, err := a.client.ExecuteCommand(ctx, "domains:remove-global", []string{domainName})
	if err != nil {
		return fmt.Errorf("failed to remove global domain %s: %w", domainName, err)
	}
	return nil
}

func (a *DokkuCoreAdapter) SetGlobalDomains(ctx context.Context, domains []string) error {
	args := domains
	_, err := a.client.ExecuteCommand(ctx, "domains:set-global", args)
	if err != nil {
		return fmt.Errorf("failed to set global domains: %w", err)
	}
	return nil
}

func (a *DokkuCoreAdapter) ClearGlobalDomains(ctx context.Context) error {
	_, err := a.client.ExecuteCommand(ctx, "domains:clear-global", []string{})
	if err != nil {
		return fmt.Errorf("failed to clear global domains: %w", err)
	}
	return nil
}

func (a *DokkuCoreAdapter) GetDomainsReport(ctx context.Context) (*domain.DomainsReport, error) {
	output, err := a.client.ExecuteCommand(ctx, "domains:report", []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to get domains report: %w", err)
	}

	// Parse the domains report output and convert to a structured format
	report := &domain.DomainsReport{
		RawOutput:   string(output),
		GeneratedAt: time.Now(),
		PerApp:      make(map[string]domain.DomainsReportSection),
	}

	// Parse the output to populate Global and PerApp sections
	// This is a basic implementation - can be enhanced with proper parsing
	lines := strings.Split(string(output), "\n")
	currentApp := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect app sections
		if strings.HasPrefix(line, "=====> ") && strings.HasSuffix(line, " domains information") {
			appName := strings.TrimPrefix(line, "=====> ")
			appName = strings.TrimSuffix(appName, " domains information")
			currentApp = appName
			report.PerApp[currentApp] = domain.DomainsReportSection{
				AppName: currentApp,
			}
		}

		// Parse global domains
		if currentApp == "" && strings.Contains(line, "Global vhosts:") {
			if idx := strings.Index(line, ":"); idx != -1 {
				domainsStr := strings.TrimSpace(line[idx+1:])
				if domainsStr != "" && domainsStr != "none" {
					report.Global.Vhosts = strings.Fields(domainsStr)
				}
			}
		}
	}

	return report, nil
}

// SSHKeyRepository implementation
func (a *DokkuCoreAdapter) ListSSHKeys(ctx context.Context) ([]domain.SSHKey, error) {
	output, err := a.client.ExecuteCommand(ctx, "ssh-keys:list", []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to list SSH keys: %w", err)
	}

	return a.parseSSHKeys(string(output)), nil
}

func (a *DokkuCoreAdapter) AddSSHKey(ctx context.Context, name string, keyContent string) error {
	// Note: Dokku SSH key addition typically requires piping input
	// For now, we'll return an error suggesting the user add keys manually
	// This could be enhanced by writing to a temporary file and using that file path
	return fmt.Errorf("SSH key addition via MCP not yet supported - please add keys manually via 'dokku ssh-keys:add %s'", name)
}

func (a *DokkuCoreAdapter) RemoveSSHKey(ctx context.Context, name string) error {
	_, err := a.client.ExecuteCommand(ctx, "ssh-keys:remove", []string{name})
	if err != nil {
		return fmt.Errorf("failed to remove SSH key %s: %w", name, err)
	}
	return nil
}

func (a *DokkuCoreAdapter) GetSSHKey(ctx context.Context, name string) (*domain.SSHKey, error) {
	keys, err := a.ListSSHKeys(ctx)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		if key.Name == name {
			return &key, nil
		}
	}

	return nil, fmt.Errorf("SSH key %s not found", name)
}

// RegistryRepository implementation
func (a *DokkuCoreAdapter) ListRegistries(ctx context.Context) ([]domain.RegistryCredential, error) {
	// Dokku doesn't have a direct command to list registries
	// We would need to implement this based on available information
	// For now, returning empty list
	return []domain.RegistryCredential{}, nil
}

func (a *DokkuCoreAdapter) LoginRegistry(ctx context.Context, registry, username, password string) error {
	// Note: Registry login typically requires password input
	// For now, we'll return an error suggesting manual login
	// This could be enhanced by using environment variables or credential files
	return fmt.Errorf("registry login via MCP not yet supported - please login manually via 'dokku registry:login %s %s'", registry, username)
}

func (a *DokkuCoreAdapter) LogoutRegistry(ctx context.Context, registry string) error {
	_, err := a.client.ExecuteCommand(ctx, "registry:logout", []string{registry})
	if err != nil {
		return fmt.Errorf("failed to logout from registry %s: %w", registry, err)
	}
	return nil
}

func (a *DokkuCoreAdapter) GetRegistryStatus(ctx context.Context, registry string) (*domain.RegistryCredential, error) {
	// This would need to be implemented based on Dokku's registry status capabilities
	return &domain.RegistryCredential{
		Registry: registry,
		Active:   false,
		AddedAt:  time.Now(),
	}, nil
}

// ConfigurationRepository implementation
func (a *DokkuCoreAdapter) GetGlobalConfiguration(ctx context.Context) (*domain.GlobalConfiguration, error) {
	config := &domain.GlobalConfiguration{
		CustomVars: make(map[string]string),
	}

	// Get proxy type
	if proxyOutput, err := a.client.ExecuteCommand(ctx, "proxy:report", []string{"--global", "--proxy-type"}); err == nil {
		config.ProxyType = strings.TrimSpace(string(proxyOutput))
	}

	// Get scheduler
	if schedulerOutput, err := a.client.ExecuteCommand(ctx, "scheduler:report", []string{"--global", "--scheduler-selected"}); err == nil {
		config.Scheduler = strings.TrimSpace(string(schedulerOutput))
	}

	// Get deploy branch
	if branchOutput, err := a.client.ExecuteCommand(ctx, "git:report", []string{"--global", "--git-deploy-branch"}); err == nil {
		config.DeployBranch = strings.TrimSpace(string(branchOutput))
	}

	return config, nil
}

func (a *DokkuCoreAdapter) SetGlobalProxyType(ctx context.Context, proxyType string) error {
	_, err := a.client.ExecuteCommand(ctx, "proxy:set", []string{"--global", proxyType})
	if err != nil {
		return fmt.Errorf("failed to set global proxy type: %w", err)
	}
	return nil
}

func (a *DokkuCoreAdapter) SetGlobalScheduler(ctx context.Context, scheduler string) error {
	_, err := a.client.ExecuteCommand(ctx, "scheduler:set", []string{"--global", "selected", scheduler})
	if err != nil {
		return fmt.Errorf("failed to set global scheduler: %w", err)
	}
	return nil
}

func (a *DokkuCoreAdapter) SetGlobalDeployBranch(ctx context.Context, branch string) error {
	_, err := a.client.ExecuteCommand(ctx, "git:set", []string{"--global", "deploy-branch", branch})
	if err != nil {
		return fmt.Errorf("failed to set global deploy branch: %w", err)
	}
	return nil
}

func (a *DokkuCoreAdapter) SetLetsEncryptEmail(ctx context.Context, email string) error {
	_, err := a.client.ExecuteCommand(ctx, "letsencrypt:set", []string{"--global", "email", email})
	if err != nil {
		return fmt.Errorf("failed to set letsencrypt email: %w", err)
	}
	return nil
}

func (a *DokkuCoreAdapter) SetVectorSink(ctx context.Context, sink string) error {
	_, err := a.client.ExecuteCommand(ctx, "logs:set", []string{"--global", "vector-sink", sink})
	if err != nil {
		return fmt.Errorf("failed to set vector sink: %w", err)
	}
	return nil
}

func (a *DokkuCoreAdapter) GetConfigurationKeys(ctx context.Context, scope string) ([]domain.ConfigurationKey, error) {
	// This would need to be implemented based on available configuration commands
	return []domain.ConfigurationKey{}, nil
}

// Helper parsing methods
func (a *DokkuCoreAdapter) parseGlobalDomains(output string) []string {
	var domains []string
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "Global vhosts:") {
			// Parse domain list from the line
			if idx := strings.Index(line, ":"); idx != -1 {
				domainsStr := strings.TrimSpace(line[idx+1:])
				if domainsStr != "" && domainsStr != "none" {
					domains = strings.Fields(domainsStr)
				}
			}
			break
		}
	}

	return domains
}

func (a *DokkuCoreAdapter) parsePluginList(output string) []domain.DokkuPlugin {
	var plugins []domain.DokkuPlugin
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "====") || strings.HasPrefix(line, "Plugin name") {
			continue
		}

		// Plugin list format: "plugin-name    version    enabled/disabled    description..."
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			plugin := domain.DokkuPlugin{
				Name:    parts[0],
				Version: parts[1],
				Status:  parts[2],
			}

			if len(parts) > 3 {
				plugin.Description = strings.Join(parts[3:], " ")
			}

			// Determine if this is a core plugin
			// Core plugins typically have descriptions containing "dokku core"
			// and have empty source fields (bundled with Dokku)
			plugin.CorePlugin = strings.Contains(strings.ToLower(plugin.Description), "dokku core") ||
				strings.Contains(strings.ToLower(plugin.Description), "core plugin")

			plugins = append(plugins, plugin)
		}
	}

	return plugins
}

func (a *DokkuCoreAdapter) parseSSHKeys(output string) []domain.SSHKey {
	var keys []domain.SSHKey
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse SSH key information
		// This is a simplified parser - real implementation would be more robust
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			key := domain.SSHKey{
				Name:        parts[0],
				Fingerprint: "",
				KeyType:     "",
				Comment:     "",
				AddedAt:     time.Now(),
			}

			if len(parts) > 1 {
				key.Fingerprint = parts[1]
			}

			keys = append(keys, key)
		}
	}

	return keys
}

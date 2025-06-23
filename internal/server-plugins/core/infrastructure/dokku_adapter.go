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

// executeCommand wraps the client's ExecuteCommand with core-specific context and validation
func (a *DokkuCoreAdapter) executeCommand(ctx context.Context, command domain.CoreCommand, args []string) ([]byte, error) {
	// Validate command is allowed
	if !command.IsValid() {
		return nil, fmt.Errorf("invalid core command: %s", command)
	}

	return a.client.ExecuteCommand(ctx, command.String(), args)
}

// SystemRepository implementation
func (a *DokkuCoreAdapter) GetSystemStatus(ctx context.Context) (*domain.SystemStatus, error) {
	status := &domain.SystemStatus{
		LastUpdated: time.Now(),
	}

	// Get version
	versionOutput, err := a.executeCommand(ctx, domain.CommandVersion, []string{})
	if err != nil {
		a.logger.Warn("Failed to get Dokku version", "error", err)
		status.Version = "unknown"
	} else {
		status.Version = strings.TrimSpace(string(versionOutput))
	}

	// Get proxy type
	proxyOutput, err := a.executeCommand(ctx, domain.CommandProxyReport, []string{"--global", "--proxy-type"})
	if err != nil {
		a.logger.Warn("Failed to get proxy type", "error", err)
		status.ProxyType = "nginx" // default
	} else {
		status.ProxyType = strings.TrimSpace(string(proxyOutput))
	}

	// Get scheduler
	schedulerOutput, err := a.executeCommand(ctx, domain.CommandSchedulerReport, []string{"--global", "--scheduler-selected"})
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
	output, err := a.executeCommand(ctx, domain.CommandPluginList, []string{})
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

	_, err := a.executeCommand(ctx, domain.CommandPluginInstall, args)
	if err != nil {
		return fmt.Errorf("failed to install plugin %s: %w", source, err)
	}

	return nil
}

func (a *DokkuCoreAdapter) UninstallPlugin(ctx context.Context, name string) error {
	_, err := a.executeCommand(ctx, domain.CommandPluginUninstall, []string{name})
	if err != nil {
		return fmt.Errorf("failed to uninstall plugin %s: %w", name, err)
	}
	return nil
}

func (a *DokkuCoreAdapter) EnablePlugin(ctx context.Context, name string) error {
	_, err := a.executeCommand(ctx, domain.CommandPluginEnable, []string{name})
	if err != nil {
		return fmt.Errorf("failed to enable plugin %s: %w", name, err)
	}
	return nil
}

func (a *DokkuCoreAdapter) DisablePlugin(ctx context.Context, name string) error {
	_, err := a.executeCommand(ctx, domain.CommandPluginDisable, []string{name})
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

	_, err := a.executeCommand(ctx, domain.CommandPluginUpdate, args)
	if err != nil {
		return fmt.Errorf("failed to update plugin %s: %w", name, err)
	}
	return nil
}

// SSHKeyRepository implementation
func (a *DokkuCoreAdapter) ListSSHKeys(ctx context.Context) ([]domain.SSHKey, error) {
	output, err := a.executeCommand(ctx, domain.CommandSSHKeysList, []string{})
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
	_, err := a.executeCommand(ctx, domain.CommandSSHKeysRemove, []string{name})
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
	_, err := a.executeCommand(ctx, domain.CommandRegistryLogout, []string{registry})
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
	if proxyOutput, err := a.executeCommand(ctx, domain.CommandProxyReport, []string{"--global", "--proxy-type"}); err == nil {
		config.ProxyType = strings.TrimSpace(string(proxyOutput))
	}

	// Get scheduler
	if schedulerOutput, err := a.executeCommand(ctx, domain.CommandSchedulerReport, []string{"--global", "--scheduler-selected"}); err == nil {
		config.Scheduler = strings.TrimSpace(string(schedulerOutput))
	}

	// Get deploy branch
	if branchOutput, err := a.executeCommand(ctx, domain.CommandGitReport, []string{"--global", "--git-deploy-branch"}); err == nil {
		config.DeployBranch = strings.TrimSpace(string(branchOutput))
	}

	return config, nil
}

func (a *DokkuCoreAdapter) SetGlobalProxyType(ctx context.Context, proxyType string) error {
	_, err := a.executeCommand(ctx, domain.CommandProxySet, []string{"--global", proxyType})
	if err != nil {
		return fmt.Errorf("failed to set global proxy type: %w", err)
	}
	return nil
}

func (a *DokkuCoreAdapter) SetGlobalScheduler(ctx context.Context, scheduler string) error {
	_, err := a.executeCommand(ctx, domain.CommandSchedulerSet, []string{"--global", "selected", scheduler})
	if err != nil {
		return fmt.Errorf("failed to set global scheduler: %w", err)
	}
	return nil
}

func (a *DokkuCoreAdapter) SetGlobalDeployBranch(ctx context.Context, branch string) error {
	_, err := a.executeCommand(ctx, domain.CommandGitSet, []string{"--global", "deploy-branch", branch})
	if err != nil {
		return fmt.Errorf("failed to set global deploy branch: %w", err)
	}
	return nil
}

func (a *DokkuCoreAdapter) SetLetsEncryptEmail(ctx context.Context, email string) error {
	_, err := a.executeCommand(ctx, domain.CommandLetsEncryptSet, []string{"--global", "email", email})
	if err != nil {
		return fmt.Errorf("failed to set letsencrypt email: %w", err)
	}
	return nil
}

func (a *DokkuCoreAdapter) SetVectorSink(ctx context.Context, sink string) error {
	_, err := a.executeCommand(ctx, domain.CommandLogsSet, []string{"--global", "vector-sink", sink})
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

func (a *DokkuCoreAdapter) parsePluginList(output string) []domain.DokkuPlugin {
	var plugins []domain.DokkuPlugin
	fieldsOutput := dokkuApi.ParseFieldsOutput(output, true)

	for _, fields := range fieldsOutput {
		if len(fields) >= 3 {
			plugin := domain.DokkuPlugin{
				Name:    fields[0],
				Version: fields[1],
				Status:  fields[2],
			}

			if len(fields) > 3 {
				plugin.Description = strings.Join(fields[3:], " ")
			}

			// Determine if this is a core plugin
			plugin.CorePlugin = strings.Contains(strings.ToLower(plugin.Description), "dokku core") ||
				strings.Contains(strings.ToLower(plugin.Description), "core plugin")

			plugins = append(plugins, plugin)
		}
	}

	return plugins
}

func (a *DokkuCoreAdapter) parseSSHKeys(output string) []domain.SSHKey {
	var keys []domain.SSHKey
	lines := dokkuApi.ParseTrimmedLines(output, true)

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			key := domain.SSHKey{
				Name:        fields[0],
				Fingerprint: "",
				KeyType:     "",
				Comment:     "",
				AddedAt:     time.Now(),
			}

			if len(fields) > 1 {
				key.Fingerprint = fields[1]
			}

			keys = append(keys, key)
		}
	}

	return keys
}

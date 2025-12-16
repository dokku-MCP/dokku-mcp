package dokkuApi

import (
	"fmt"
	"log/slog"
	"os/exec"
	"time"

	"github.com/dokku-mcp/dokku-mcp/pkg/config"
)

// SSHConnectionManager combines SSH configuration with authentication management
type SSHConnectionManager struct {
	config      *SSHConfig
	authService *SSHAuthService
	logger      *slog.Logger
}

// NewSSHConnectionManager creates a new SSH connection manager
func NewSSHConnectionManager(config *SSHConfig, logger *slog.Logger) *SSHConnectionManager {
	return &SSHConnectionManager{
		config:      config,
		authService: NewSSHAuthService(logger),
		logger:      logger,
	}
}

// NewSSHConnectionManagerFromServerConfig creates an SSH connection manager from server configuration
func NewSSHConnectionManagerFromServerConfig(cfg *config.ServerConfig, logger *slog.Logger) (*SSHConnectionManager, error) {
	sshConfig, err := NewSSHConfigFromServerConfig(
		cfg.SSH.Host,
		cfg.SSH.Port,
		cfg.SSH.User,
		cfg.SSH.KeyPath,
		cfg.Timeout,
		cfg.SSH.DisablePTY,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH configuration: %w", err)
	}

	return NewSSHConnectionManager(sshConfig, logger), nil
}

// Config returns the SSH configuration
func (m *SSHConnectionManager) Config() *SSHConfig {
	return m.config
}

// UpdateConfig updates the SSH configuration
func (m *SSHConnectionManager) UpdateConfig(newConfig *SSHConfig) {
	m.config = newConfig
}

// PrepareSSHCommand prepares a complete SSH command with authentication
func (m *SSHConnectionManager) PrepareSSHCommand(command string) ([]string, []string, error) {
	// Determine the best authentication method
	authMethod := m.authService.DetermineAuthMethod(m.config.KeyPath())

	// Start with base SSH arguments
	sshArgs := []string{"ssh"}
	sshArgs = append(sshArgs, m.config.BaseSSHArgs()...)

	// Apply authentication method
	sshArgs = m.authService.PrepareSSHArgs(authMethod, sshArgs)

	// Add destination
	sshArgs = append(sshArgs, m.config.ConnectionString())

	// Add command if specified
	if command != "" {
		sshArgs = append(sshArgs, "--", command)
	}

	// Prepare environment
	baseEnv := []string{
		"PATH=/usr/bin:/bin",
		fmt.Sprintf("DOKKU_HOST=%s", m.config.Host()),
		fmt.Sprintf("DOKKU_PORT=%d", m.config.Port()),
	}
	env := m.authService.PrepareEnvironment(authMethod, baseEnv)

	m.logger.Debug("Prepared SSH command",
		"ssh_args", sshArgs,
		"auth_method", authMethod.Description,
		"target", m.config.ConnectionString())

	return sshArgs, env, nil
}

// TestConnection tests the SSH connection
func (m *SSHConnectionManager) TestConnection() error {
	sshArgs, env, err := m.PrepareSSHCommand("echo 'connection_test'")
	if err != nil {
		return fmt.Errorf("failed to prepare SSH command: %w", err)
	}

	// #nosec G204 -- Integration test, not a user command
	cmd := exec.Command(sshArgs[0], sshArgs[1:]...)
	cmd.Env = env

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("SSH connection test failed: %w", err)
	}

	m.logger.Debug("SSH connection test successful", "output", string(output))

	// Mark configuration as verified
	m.config = m.config.MarkAsVerified()

	return nil
}

// ConnectionInfo represents SSH connection information
type ConnectionInfo struct {
	Host             string        `json:"host"`
	Port             int           `json:"port"`
	User             string        `json:"user"`
	KeyPath          string        `json:"key_path"`
	Timeout          time.Duration `json:"timeout"`
	Verified         bool          `json:"verified"`
	AuthMethod       string        `json:"auth_method"`
	ConnectionString string        `json:"connection_string"`
}

// GetConnectionInfo returns human-readable connection information
func (m *SSHConnectionManager) GetConnectionInfo() ConnectionInfo {
	authMethod := m.authService.DetermineAuthMethod(m.config.KeyPath())

	return ConnectionInfo{
		Host:             m.config.Host(),
		Port:             m.config.Port(),
		User:             m.config.User(),
		KeyPath:          m.config.KeyPath(),
		Timeout:          m.config.Timeout(),
		Verified:         m.config.IsVerified(),
		AuthMethod:       authMethod.Description,
		ConnectionString: m.config.ConnectionString(),
	}
}

// SSHConfigBuilder provides a fluent interface for building SSH configurations
type SSHConfigBuilder struct {
	host       string
	port       int
	user       string
	keyPath    string
	timeout    time.Duration
	disablePTY bool
	logger     *slog.Logger
}

// NewSSHConfigBuilder creates a new SSH configuration builder
func NewSSHConfigBuilder(logger *slog.Logger) *SSHConfigBuilder {
	return &SSHConfigBuilder{
		port:    22,
		timeout: 30 * time.Second,
		logger:  logger,
	}
}

// WithHost sets the SSH host
func (b *SSHConfigBuilder) WithHost(host string) *SSHConfigBuilder {
	b.host = host
	return b
}

// WithPort sets the SSH port
func (b *SSHConfigBuilder) WithPort(port int) *SSHConfigBuilder {
	b.port = port
	return b
}

// WithUser sets the SSH user
func (b *SSHConfigBuilder) WithUser(user string) *SSHConfigBuilder {
	b.user = user
	return b
}

// WithKeyPath sets the SSH key path
func (b *SSHConfigBuilder) WithKeyPath(keyPath string) *SSHConfigBuilder {
	b.keyPath = keyPath
	return b
}

// WithTimeout sets the connection timeout
func (b *SSHConfigBuilder) WithTimeout(timeout time.Duration) *SSHConfigBuilder {
	b.timeout = timeout
	return b
}

// WithDisablePTY sets whether to disable PTY allocation
func (b *SSHConfigBuilder) WithDisablePTY(disable bool) *SSHConfigBuilder {
	b.disablePTY = disable
	return b
}

// FromServerConfig populates the builder from server configuration
func (b *SSHConfigBuilder) FromServerConfig(cfg *config.ServerConfig) *SSHConfigBuilder {
	return b.WithHost(cfg.SSH.Host).
		WithPort(cfg.SSH.Port).
		WithUser(cfg.SSH.User).
		WithKeyPath(cfg.SSH.KeyPath).
		WithTimeout(cfg.Timeout).
		WithDisablePTY(cfg.SSH.DisablePTY)
}

// Build creates the SSH configuration
func (b *SSHConfigBuilder) Build() (*SSHConfig, error) {
	return NewSSHConfigFromServerConfig(b.host, b.port, b.user, b.keyPath, b.timeout, b.disablePTY)
}

// BuildConnectionManager creates a complete SSH connection manager
func (b *SSHConfigBuilder) BuildConnectionManager() (*SSHConnectionManager, error) {
	config, err := b.Build()
	if err != nil {
		return nil, err
	}

	return NewSSHConnectionManager(config, b.logger), nil
}

package dokku

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type SSHAuthMethod struct {
	KeyPath     string
	UseAgent    bool
	Description string
}

type SSHAuthConfig struct {
	// HomeDir allows to override the home directory for tests
	HomeDir string
	// CheckAgent allows to override the ssh-agent check for tests
	CheckAgent func() bool
}

type SSHAuthService struct {
	logger *slog.Logger
	config *SSHAuthConfig
}

func NewSSHAuthService(logger *slog.Logger) *SSHAuthService {
	return &SSHAuthService{
		logger: logger,
		config: &SSHAuthConfig{},
	}
}

func NewSSHAuthServiceWithConfig(logger *slog.Logger, config *SSHAuthConfig) *SSHAuthService {
	return &SSHAuthService{
		logger: logger,
		config: config,
	}
}

// Priority: 1. ssh-agent, 2. ~/.ssh/id_rsa fallback, 3. configured key
func (s *SSHAuthService) DetermineAuthMethod(configKeyPath string) *SSHAuthMethod {
	// 1. Try ssh-agent first (default)
	if s.checkSshAgentAvailable() {
		s.logger.Debug("Using ssh-agent for authentication")
		return &SSHAuthMethod{
			UseAgent:    true,
			Description: "ssh-agent",
		}
	}

	// 2. Fallback to ~/.ssh/id_rsa if the file exists
	homeDir := s.getHomeDir()
	if homeDir != "" {
		defaultKeyPath := filepath.Join(homeDir, ".ssh", "id_rsa")
		if s.isKeyFileAccessible(defaultKeyPath) {
			s.logger.Debug("Using default SSH key",
				"key_path", defaultKeyPath)
			return &SSHAuthMethod{
				KeyPath:     defaultKeyPath,
				Description: "default key ~/.ssh/id_rsa",
			}
		}
	}

	// 3. Use the configured key if specified
	if configKeyPath != "" {
		if s.isKeyFileAccessible(configKeyPath) {
			s.logger.Debug("Using the configured SSH key",
				"key_path", configKeyPath)
			return &SSHAuthMethod{
				KeyPath:     configKeyPath,
				Description: fmt.Sprintf("configured key %s", configKeyPath),
			}
		} else {
			s.logger.Warn("The configured SSH key is not accessible",
				"key_path", configKeyPath)
		}
	}

	// No method available - use ssh-agent as fallback
	s.logger.Warn("No reliable SSH authentication method found, using ssh-agent as fallback")
	return &SSHAuthMethod{
		UseAgent:    true,
		Description: "ssh-agent (fallback)",
	}
}

func (s *SSHAuthService) getHomeDir() string {
	if s.config.HomeDir != "" {
		return s.config.HomeDir
	}
	homeDir, _ := os.UserHomeDir()
	return homeDir
}

func (s *SSHAuthService) checkSshAgentAvailable() bool {
	if s.config.CheckAgent != nil {
		return s.config.CheckAgent()
	}
	return s.isSshAgentAvailable()
}

func (s *SSHAuthService) isSshAgentAvailable() bool {
	authSock := os.Getenv("SSH_AUTH_SOCK")
	if authSock == "" {
		s.logger.Debug("SSH_AUTH_SOCK is not defined - ssh-agent is not available")
		return false
	}

	if _, err := os.Stat(authSock); os.IsNotExist(err) {
		s.logger.Debug("Socket ssh-agent does not exist",
			"socket_path", authSock)
		return false
	}

	// Test if ssh-add can list the keys
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ssh-add", "-l")
	cmd.Env = []string{
		fmt.Sprintf("SSH_AUTH_SOCK=%s", authSock),
		"PATH=/usr/bin:/bin",
	}

	output, err := cmd.Output()
	if err != nil {
		s.logger.Debug("ssh-add -l failed",
			"error", err)
		return false
	}

	// If the output contains "no identities", ssh-agent works but has no keys loaded
	outputStr := string(output)
	if strings.Contains(outputStr, "no identities") {
		s.logger.Debug("ssh-agent works but has no keys loaded")
		return false
	}

	s.logger.Debug("ssh-agent is available and has keys loaded")
	return true
}

func (s *SSHAuthService) isKeyFileAccessible(keyPath string) bool {
	if keyPath == "" {
		return false
	}

	// Expand tilde in path
	if strings.HasPrefix(keyPath, "~/") {
		homeDir := s.getHomeDir()
		if homeDir == "" {
			return false
		}
		keyPath = filepath.Join(homeDir, keyPath[2:])
	}

	// Check if file exists and is readable
	info, err := os.Stat(keyPath)
	if err != nil {
		return false
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return false
	}

	// Try to open the file to verify read access
	file, err := os.Open(keyPath)
	if err != nil {
		return false
	}
	file.Close()

	return true
}

func (s *SSHAuthService) PrepareSSHArgs(authMethod *SSHAuthMethod, baseArgs []string) []string {
	if !authMethod.UseAgent && authMethod.KeyPath != "" {
		return append([]string{"-i", authMethod.KeyPath}, baseArgs...)
	}
	return baseArgs
}

func (s *SSHAuthService) PrepareEnvironment(authMethod *SSHAuthMethod, baseEnv []string) []string {
	env := make([]string, len(baseEnv))
	copy(env, baseEnv)

	// Add SSH_AUTH_SOCK if using ssh-agent
	if authMethod.UseAgent {
		if authSock := os.Getenv("SSH_AUTH_SOCK"); authSock != "" {
			env = append(env, fmt.Sprintf("SSH_AUTH_SOCK=%s", authSock))
		}
	}

	return env
}

package dokkuApi

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
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

	// Authentication caching
	cachedAuthMethod *SSHAuthMethod
	cacheExpiry      time.Time
	cacheMutex       sync.RWMutex
	cacheTTL         time.Duration
}

func NewSSHAuthService(logger *slog.Logger) *SSHAuthService {
	return &SSHAuthService{
		logger:   logger,
		config:   &SSHAuthConfig{},
		cacheTTL: 60 * time.Minute,
	}
}

func NewSSHAuthServiceWithConfig(logger *slog.Logger, config *SSHAuthConfig) *SSHAuthService {
	return &SSHAuthService{
		logger:   logger,
		config:   config,
		cacheTTL: 60 * time.Minute,
	}
}

// Priority: 1. configured key, 2. ssh-agent, 3. ~/.ssh/id_rsa fallback
func (s *SSHAuthService) DetermineAuthMethod(configKeyPath string) *SSHAuthMethod {
	// Check cache first
	s.cacheMutex.RLock()
	if s.cachedAuthMethod != nil && time.Now().Before(s.cacheExpiry) {
		method := s.cachedAuthMethod
		s.cacheMutex.RUnlock()
		s.logger.Debug("Using cached SSH auth method", "method", method.Description)
		return method
	}
	s.cacheMutex.RUnlock()

	// Determine new auth method
	method := s.determineAuthMethodUncached(configKeyPath)

	// Cache the result
	s.cacheMutex.Lock()
	s.cachedAuthMethod = method
	s.cacheExpiry = time.Now().Add(s.cacheTTL)
	s.cacheMutex.Unlock()

	return method
}

func (s *SSHAuthService) determineAuthMethodUncached(configKeyPath string) *SSHAuthMethod {
	// 1. Use the configured key if specified and accessible (highest priority)
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

	// 2. Try ssh-agent if available (second priority)
	if s.checkSshAgentAvailable() {
		s.logger.Debug("Using ssh-agent for authentication")
		return &SSHAuthMethod{
			UseAgent:    true,
			Description: "ssh-agent",
		}
	}

	// 3. Fallback to ~/.ssh/id_rsa if the file exists
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

	// No method available - use ssh-agent as fallback
	s.logger.Warn("No reliable SSH authentication method found, using ssh-agent as fallback")
	return &SSHAuthMethod{
		UseAgent:    true,
		Description: "ssh-agent (fallback)",
	}
}

// InvalidateCache forces a re-check of authentication method on next call
func (s *SSHAuthService) InvalidateCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.cachedAuthMethod = nil
	s.cacheExpiry = time.Time{}
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

	// Clean the keyPath to prevent path traversal
	keyPath = filepath.Clean(keyPath)

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
	// Handle error from file.Close() (G104)
	if cerr := file.Close(); cerr != nil {
		s.logger.Warn("Error closing key file", "error", cerr)
		return false
	}

	return true
}

func (s *SSHAuthService) PrepareSSHArgs(authMethod *SSHAuthMethod, baseArgs []string) []string {
	if !authMethod.UseAgent && authMethod.KeyPath != "" {
		// Insert -i and key path after the first argument (ssh command)
		if len(baseArgs) > 0 {
			result := make([]string, 0, len(baseArgs)+2)
			result = append(result, baseArgs[0])              // ssh command
			result = append(result, "-i", authMethod.KeyPath) // add -i flag and key
			result = append(result, baseArgs[1:]...)          // add remaining args
			return result
		}
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

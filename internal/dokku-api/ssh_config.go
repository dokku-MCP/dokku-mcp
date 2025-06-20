package dokkuApi

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// SSHConfig represents a validated and secure SSH configuration for Dokku
type SSHConfig struct {
	host     string
	port     int
	user     string
	keyPath  string
	timeout  time.Duration
	verified bool
}

var (
	// Pattern to validate hostnames
	hostnamePattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.-]{0,61}[a-zA-Z0-9])?$`)
	// Pattern to validate SSH usernames
	usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

// NewSSHConfig creates a new SSH configuration with validation
func NewSSHConfig(host string, port int, user string, keyPath string, timeout time.Duration) (*SSHConfig, error) {
	host = strings.TrimSpace(host)
	user = strings.TrimSpace(user)
	keyPath = strings.TrimSpace(keyPath)

	if err := validateSSHConfig(host, port, user, keyPath, timeout); err != nil {
		return nil, fmt.Errorf("invalid SSH configuration: %w", err)
	}

	return &SSHConfig{
		host:     host,
		port:     port,
		user:     user,
		keyPath:  keyPath,
		timeout:  timeout,
		verified: false,
	}, nil
}

// NewDefaultSSHConfig creates a default SSH configuration for Dokku
func NewDefaultSSHConfig() *SSHConfig {
	return &SSHConfig{
		host:    "pro.dokku.com",
		port:    22,
		user:    "dokku",
		keyPath: "", // Uses SSH agent or default key
		timeout: 30 * time.Second,
	}
}

// NewSSHConfigFromServerConfig creates an SSHConfig from ServerConfig parameters
func NewSSHConfigFromServerConfig(dokkuHost string, dokkuPort int, dokkuUser string, sshKeyPath string, timeout time.Duration) (*SSHConfig, error) {
	return NewSSHConfig(dokkuHost, dokkuPort, dokkuUser, sshKeyPath, timeout)
}

// MustNewSSHConfig creates an SSH configuration, panicking on error
func MustNewSSHConfig(host string, port int, user string, keyPath string, timeout time.Duration) *SSHConfig {
	config, err := NewSSHConfig(host, port, user, keyPath, timeout)
	if err != nil {
		panic(fmt.Sprintf("failed to create SSH configuration: %v", err))
	}
	return config
}

// Host returns the SSH host
func (s *SSHConfig) Host() string {
	return s.host
}

// Port returns the SSH port
func (s *SSHConfig) Port() int {
	return s.port
}

// User returns the SSH user
func (s *SSHConfig) User() string {
	return s.user
}

// KeyPath returns the path to the SSH key
func (s *SSHConfig) KeyPath() string {
	return s.keyPath
}

// Timeout returns the connection timeout
func (s *SSHConfig) Timeout() time.Duration {
	return s.timeout
}

// IsVerified returns whether the configuration has been verified
func (s *SSHConfig) IsVerified() bool {
	return s.verified
}

// ConnectionString returns the SSH connection string
func (s *SSHConfig) ConnectionString() string {
	return fmt.Sprintf("%s@%s", s.user, s.host)
}

// BaseSSHArgs returns the base SSH command arguments
func (s *SSHConfig) BaseSSHArgs() []string {
	return []string{
		"-o", "LogLevel=QUIET",
		"-o", "StrictHostKeyChecking=no",
		"-o", fmt.Sprintf("ConnectTimeout=%d", int(s.timeout.Seconds())),
		"-p", fmt.Sprintf("%d", s.port),
	}
}

// SSHCommand returns the complete SSH command
func (s *SSHConfig) SSHCommand(command string) []string {
	args := []string{"ssh"}
	args = append(args, s.BaseSSHArgs()...)

	// Add SSH key if specified
	if s.keyPath != "" {
		args = append(args, "-i", s.keyPath)
	}

	// Add destination
	args = append(args, s.ConnectionString())

	// Add command
	if command != "" {
		args = append(args, "--", command)
	}

	return args
}

// WithHost returns a new configuration with a different host
func (s *SSHConfig) WithHost(host string) (*SSHConfig, error) {
	return NewSSHConfig(host, s.port, s.user, s.keyPath, s.timeout)
}

// WithPort returns a new configuration with a different port
func (s *SSHConfig) WithPort(port int) (*SSHConfig, error) {
	return NewSSHConfig(s.host, port, s.user, s.keyPath, s.timeout)
}

// WithUser returns a new configuration with a different user
func (s *SSHConfig) WithUser(user string) (*SSHConfig, error) {
	return NewSSHConfig(s.host, s.port, user, s.keyPath, s.timeout)
}

// WithKeyPath returns a new configuration with a different key path
func (s *SSHConfig) WithKeyPath(keyPath string) (*SSHConfig, error) {
	return NewSSHConfig(s.host, s.port, s.user, keyPath, s.timeout)
}

// WithTimeout returns a new configuration with a different timeout
func (s *SSHConfig) WithTimeout(timeout time.Duration) (*SSHConfig, error) {
	return NewSSHConfig(s.host, s.port, s.user, s.keyPath, timeout)
}

// IsLocalhost checks if the host is localhost
func (s *SSHConfig) IsLocalhost() bool {
	return s.host == "localhost" || s.host == "127.0.0.1" || s.host == "::1"
}

// UsesDefaultKey checks if the configuration uses the default key
func (s *SSHConfig) UsesDefaultKey() bool {
	return s.keyPath == ""
}

// HasCustomKey checks if a custom key is specified
func (s *SSHConfig) HasCustomKey() bool {
	return s.keyPath != ""
}

// KeyExists checks if the key file exists (if specified)
func (s *SSHConfig) KeyExists() bool {
	if s.keyPath == "" {
		return true // No key specified, assume SSH agent or default key works
	}

	if _, err := os.Stat(s.keyPath); err != nil {
		return false
	}
	return true
}

// GetKeyFingerprint returns the key fingerprint (if possible)
func (s *SSHConfig) GetKeyFingerprint() (string, error) {
	if s.keyPath == "" {
		return "", fmt.Errorf("no key path specified")
	}

	if !s.KeyExists() {
		return "", fmt.Errorf("key file does not exist: %s", s.keyPath)
	}

	// In practice, we would use ssh-keygen to get the fingerprint
	// For now, we return just the path
	return fmt.Sprintf("key:%s", filepath.Base(s.keyPath)), nil
}

// Validate performs complete configuration validation
func (s *SSHConfig) Validate() error {
	return validateSSHConfig(s.host, s.port, s.user, s.keyPath, s.timeout)
}

// MarkAsVerified marks the configuration as verified
func (s *SSHConfig) MarkAsVerified() *SSHConfig {
	// Returns a new instance with verified=true (immutability)
	return &SSHConfig{
		host:     s.host,
		port:     s.port,
		user:     s.user,
		keyPath:  s.keyPath,
		timeout:  s.timeout,
		verified: true,
	}
}

// String implements fmt.Stringer
func (s *SSHConfig) String() string {
	if s.keyPath != "" {
		return fmt.Sprintf("ssh://%s@%s:%d (key: %s)", s.user, s.host, s.port, filepath.Base(s.keyPath))
	}
	return fmt.Sprintf("ssh://%s@%s:%d", s.user, s.host, s.port)
}

// Equal compares two SSH configurations
func (s *SSHConfig) Equal(other *SSHConfig) bool {
	if other == nil {
		return false
	}
	return s.host == other.host &&
		s.port == other.port &&
		s.user == other.user &&
		s.keyPath == other.keyPath &&
		s.timeout == other.timeout
}

// validateSSHConfig validates SSH configuration parameters
func validateSSHConfig(host string, port int, user string, keyPath string, timeout time.Duration) error {
	// Host validation
	if host == "" {
		return fmt.Errorf("SSH host cannot be empty")
	}

	if len(host) > 255 {
		return fmt.Errorf("SSH host is too long (max 255 characters)")
	}

	// Validation for localhost and IPs are OK
	if host != "localhost" && host != "127.0.0.1" && host != "::1" {
		if !hostnamePattern.MatchString(host) {
			return fmt.Errorf("invalid SSH host format: %s", host)
		}
	}

	// Port validation
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid SSH port: %d (must be between 1 and 65535)", port)
	}

	// User validation
	if user == "" {
		return fmt.Errorf("SSH user cannot be empty")
	}

	if len(user) > 32 {
		return fmt.Errorf("SSH user is too long (max 32 characters)")
	}

	if !usernamePattern.MatchString(user) {
		return fmt.Errorf("invalid SSH user format: %s", user)
	}

	// Key path validation (optional)
	if keyPath != "" {
		if len(keyPath) > 4096 {
			return fmt.Errorf("SSH key path is too long (max 4096 characters)")
		}

		// Basic security checks
		if strings.Contains(keyPath, "..") {
			return fmt.Errorf("SSH key path cannot contain '..'")
		}
	}

	// Timeout validation
	if timeout < 0 {
		return fmt.Errorf("SSH timeout cannot be negative")
	}

	if timeout > 10*time.Minute {
		return fmt.Errorf("SSH timeout is too long (max 10 minutes)")
	}

	return nil
}

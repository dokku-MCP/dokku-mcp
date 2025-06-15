package dokkutesting

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

// TestConfig contains centralized configuration for integration tests
type TestConfig struct {
	// Test mode
	UseMockClient bool // Use mocks instead of real server

	// Dokku SSH Configuration
	DokkuHost  string
	DokkuPort  int
	DokkuUser  string
	SSHKeyPath string
	LogLevel   slog.Level

	// Timeouts and delays
	CommandTimeout   time.Duration
	AppCreateTimeout time.Duration
	CleanupTimeout   time.Duration

	// Prefixes and naming
	TestAppPrefix string
	MaxTestApps   int

	// Test options
	SkipSlowTests bool
	KeepTestApps  bool
	VerboseOutput bool

	// CI/CD Configuration
	IsCIEnvironment bool
	CIProvider      string

	// Dokku Features
	EventsEnabled    bool
	GitReportEnabled bool

	// Factories for dependency injection
	ClientFactory            DokkuClientFactory
	DeploymentServiceFactory DeploymentServiceFactory
}

// DefaultTestConfig returns the default configuration for tests
func DefaultTestConfig() *TestConfig {
	// Automatically detect mock mode if no Dokku server is configured
	useMock := getBoolEnv("DOKKU_MCP_USE_MOCK", true) // Default to using mocks

	config := &TestConfig{
		UseMockClient:    useMock,
		DokkuHost:        getEnv("DOKKU_HOST", "dokku.me"),
		DokkuPort:        getIntEnv("DOKKU_PORT", 22),
		DokkuUser:        getEnv("DOKKU_USER", "dokku"),
		SSHKeyPath:       getEnv("SSH_KEY_PATH", "~/.ssh/id_rsa"),
		LogLevel:         parseLogLevel(getEnv("LOG_LEVEL", "info")),
		CommandTimeout:   getDurationEnv("COMMAND_TIMEOUT", 30*time.Second),
		AppCreateTimeout: getDurationEnv("APP_CREATE_TIMEOUT", 2*time.Minute),
		CleanupTimeout:   getDurationEnv("CLEANUP_TIMEOUT", 5*time.Minute),
		TestAppPrefix:    getEnv("TEST_APP_PREFIX", "test-dokku-mcp"),
		MaxTestApps:      getIntEnv("MAX_TEST_APPS", 10),
		SkipSlowTests:    getBoolEnv("SKIP_SLOW_TESTS", false),
		KeepTestApps:     getBoolEnv("KEEP_TEST_APPS", false),
		VerboseOutput:    getBoolEnv("VERBOSE_OUTPUT", false),
		IsCIEnvironment:  detectCIEnvironment(),
		CIProvider:       detectCIProvider(),
	}

	// Adjust timeouts in CI environment
	if config.IsCIEnvironment {
		config.CommandTimeout *= 2
		config.AppCreateTimeout *= 2
		config.LogLevel = slog.LevelDebug
	}

	// Configure factories based on mode
	if config.UseMockClient {
		config.ClientFactory = &MockClientFactory{}
		config.DeploymentServiceFactory = &MockDeploymentServiceFactory{}
		config.EventsEnabled = true    // Mocks always support events
		config.GitReportEnabled = true // Mocks always support git:report
	}

	return config
}

// LoadTestConfig loads configuration from environment
func LoadTestConfig() *TestConfig {
	return DefaultTestConfig()
}

// SetFactories configures factories for dependency injection
func (c *TestConfig) SetFactories(clientFactory DokkuClientFactory, serviceFactory DeploymentServiceFactory) {
	c.ClientFactory = clientFactory
	c.DeploymentServiceFactory = serviceFactory
}

// detectDokkuFeatures detects which Dokku features are available
func (c *TestConfig) detectDokkuFeatures(client DokkuClient) {
	ctx := context.Background()

	// Test events
	_, err := client.ExecuteCommand(ctx, "events", []string{})
	c.EventsEnabled = (err == nil)

	// Test git:report with a fake app (will fail but tells us if the command exists)
	_, err = client.ExecuteCommand(ctx, "git:report", []string{"nonexistent-app-test"})
	c.GitReportEnabled = (err != nil && !isCommandNotFoundError(err))
}

// CreateLogger creates a logger configured for tests
func (c *TestConfig) CreateLogger() *slog.Logger {
	var handler slog.Handler

	handlerOpts := &slog.HandlerOptions{
		Level: c.LogLevel,
	}

	if c.VerboseOutput {
		// Even if verbose, use stderr to respect the MCP protocol
		handler = slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level:     c.LogLevel,
			AddSource: true,
		})
	} else {
		handler = slog.NewTextHandler(os.Stderr, handlerOpts)
	}

	return slog.New(handler)
}

// CreateDokkuClient creates a Dokku client configured for tests
func (c *TestConfig) CreateDokkuClient() DokkuClient {
	if c.ClientFactory == nil {
		panic("ClientFactory is not configured - use SetFactories()")
	}

	logger := c.CreateLogger()
	config := &ClientConfig{
		DokkuHost:      c.DokkuHost,
		DokkuPort:      c.DokkuPort,
		DokkuUser:      c.DokkuUser,
		SSHKeyPath:     c.SSHKeyPath,
		CommandTimeout: c.CommandTimeout,
		AllowedCommands: map[string]bool{
			"apps:list":    true,
			"apps:info":    true,
			"apps:create":  true,
			"apps:exists":  true,
			"apps:destroy": true,
			"config:get":   true,
			"config:set":   true,
			"config:show":  true,
			"events":       true,
			"git:report":   true,
		},
	}

	client := c.ClientFactory.CreateClient(config, logger)

	// Detect available Dokku features
	c.detectDokkuFeatures(client)

	return client
}

// ShouldSkipTest determines if a test should be skipped
func (c *TestConfig) ShouldSkipTest(testType TestType) bool {
	switch testType {
	case TestTypeSlow:
		return c.SkipSlowTests
	case TestTypeEvents:
		return !c.EventsEnabled
	case TestTypeGitReport:
		return !c.GitReportEnabled
	default:
		return false
	}
}

// TestType represents different types of tests
type TestType string

const (
	TestTypeFast      TestType = "fast"
	TestTypeSlow      TestType = "slow"
	TestTypeEvents    TestType = "events"
	TestTypeGitReport TestType = "gitreport"
	TestTypeBench     TestType = "benchmark"
)

// Utility functions for parsing environment

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		i, err := strconv.Atoi(value)
		if err != nil {
			return defaultValue
		}
		return i
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		duration, err := time.ParseDuration(value)
		if err != nil {
			return defaultValue
		}
		return duration
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		b, err := strconv.ParseBool(value)
		if err != nil {
			return defaultValue
		}
		return b
	}
	return defaultValue
}

func detectCIEnvironment() bool {
	ciEnvVars := []string{
		"CI", "CONTINUOUS_INTEGRATION",
		"GITHUB_ACTIONS", "GITLAB_CI", "CIRCLECI",
		"TRAVIS", "JENKINS_URL", "BUILDKITE",
	}

	for _, env := range ciEnvVars {
		if os.Getenv(env) != "" {
			return true
		}
	}

	return false
}

func detectCIProvider() string {
	providers := map[string]string{
		"GITHUB_ACTIONS": "github-actions",
		"GITLAB_CI":      "gitlab-ci",
		"CIRCLECI":       "circleci",
		"TRAVIS":         "travis",
		"JENKINS_URL":    "jenkins",
		"BUILDKITE":      "buildkite",
	}

	for env, provider := range providers {
		if os.Getenv(env) != "" {
			return provider
		}
	}

	return "unknown"
}

func isCommandNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errorMsg := err.Error()
	return fmt.Sprintf("%v", errorMsg) == "command not found" ||
		fmt.Sprintf("%v", errorMsg) == "No such plugin"
}

// GenerateTestSuffix generates a unique test suffix based on timestamp
func GenerateTestSuffix() string {
	return time.Now().Format("20060102150405")
}

// parseLogLevel converts a string to slog level
func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
}

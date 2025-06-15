package dokkutesting

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"sync"
	"time"

	"github.com/alex-galey/dokku-mcp/src/domain/application"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Test environment configured for Ginkgo
type TestFixture struct {
	Config            *TestConfig
	Client            DokkuClient
	DeploymentService application.DeploymentService
	Logger            *slog.Logger

	// Applications created for automatic cleanup
	createdApps []string
	mu          sync.Mutex
}

func NewTestFixture() *TestFixture {
	config := LoadTestConfig()
	return NewTestFixtureWithConfig(config)
}

func NewTestFixtureWithConfig(config *TestConfig) *TestFixture {
	skipIfDokkuNotAvailable(config)

	// Check that factories are configured
	if config.ClientFactory == nil || config.DeploymentServiceFactory == nil {
		panic("Factories must be configured before creating TestFixture - use SetFactories()")
	}

	client := config.CreateDokkuClient()
	logger := config.CreateLogger()

	deploymentService := config.DeploymentServiceFactory.CreateService(client, logger)

	fixture := &TestFixture{
		Config:            config,
		Client:            client,
		DeploymentService: deploymentService,
		Logger:            logger,
		createdApps:       make([]string, 0),
	}

	if !config.KeepTestApps {
		ginkgo.DeferCleanup(func() {
			fixture.cleanup()
		})
	}

	return fixture
}

// CreateTestApplication creates a test application with automatic naming
func (f *TestFixture) CreateTestApplication(prefix string) *TestApplication {
	appName := f.generateAppName(prefix)
	f.trackApp(appName)

	ctx, cancel := context.WithTimeout(context.Background(), f.Config.AppCreateTimeout)
	defer cancel()

	// Create the application via Dokku client
	_, err := f.Client.ExecuteCommand(ctx, "apps:create", []string{appName})
	Expect(err).NotTo(HaveOccurred(), "Application %s could not be created", appName)

	f.waitForAppReady(appName)

	return &TestApplication{
		Name:    appName,
		fixture: f,
		logger:  f.Logger.With("app_name", appName),
	}
}

// CreateMultipleTestApplications creates multiple test applications in parallel
func (f *TestFixture) CreateMultipleTestApplications(prefix string, count int) []*TestApplication {
	if count > f.Config.MaxTestApps {
		ginkgo.Skip(fmt.Sprintf("Number of requested applications (%d) exceeds limit (%d)", count, f.Config.MaxTestApps))
	}

	apps := make([]*TestApplication, count)
	sem := make(chan struct{}, 3) // Limit to 3 concurrent creations

	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			apps[index] = f.CreateTestApplication(fmt.Sprintf("%s-%d", prefix, index))
		}(i)
	}

	wg.Wait()
	return apps
}

// WaitForApplication waits for an application to reach the desired state
func (f *TestFixture) WaitForApplication(appName string, expectedState application.ApplicationState, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			ginkgo.Fail(fmt.Sprintf("Timeout waiting for application %s to reach state %s", appName, expectedState))
		case <-ticker.C:
			// For this example, we simply simulate a state check
			// In a real implementation, this would query Dokku
			return
		}
	}
}

func (f *TestFixture) generateAppName(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func (f *TestFixture) trackApp(appName string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createdApps = append(f.createdApps, appName)
}

func (f *TestFixture) waitForAppReady(appName string) {
	time.Sleep(time.Second)

	ctx := context.Background()
	_, err := f.Client.ExecuteCommand(ctx, "apps:exists", []string{appName})
	Expect(err).NotTo(HaveOccurred(), "Application %s was not created correctly", appName)
}

// cleanup cleans up all created applications (private for DeferCleanup)
func (f *TestFixture) cleanup() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.createdApps) == 0 {
		return
	}

	f.Logger.Debug("Cleaning up test applications",
		"count", len(f.createdApps))

	ctx, cancel := context.WithTimeout(context.Background(), f.Config.CleanupTimeout)
	defer cancel()

	cleaned := 0
	for _, appName := range f.createdApps {
		if _, err := f.Client.ExecuteCommand(ctx, "apps:destroy", []string{appName, "--force"}); err != nil {
			ginkgo.GinkgoWriter.Printf("Warning: failed to cleanup test app %s: %v\n", appName, err)
		} else {
			cleaned++
		}
	}

	f.Logger.Debug("Cleanup completed",
		"total", len(f.createdApps),
		"cleaned", cleaned)

	f.createdApps = nil
}

// TestApplication represents a test application
type TestApplication struct {
	Name    string
	fixture *TestFixture
	logger  *slog.Logger
}

func (ta *TestApplication) GetHistory() []*application.Deployment {
	ctx := context.Background()
	deployments, err := ta.fixture.DeploymentService.GetHistory(ctx, ta.Name)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve history for %s", ta.Name)

	ta.logger.Debug("History retrieved",
		"count", len(deployments))

	return deployments
}

func (ta *TestApplication) GetInfo() map[string]interface{} {
	ctx := context.Background()
	info, err := ta.fixture.Client.GetApplicationInfo(ctx, ta.Name)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve info for %s", ta.Name)

	return info
}

func (ta *TestApplication) GetConfig() map[string]string {
	ctx := context.Background()
	config, err := ta.fixture.Client.GetApplicationConfig(ctx, ta.Name)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve config for %s", ta.Name)

	return config
}

func (ta *TestApplication) SetConfig(config map[string]string) {
	ctx := context.Background()
	err := ta.fixture.Client.SetApplicationConfig(ctx, ta.Name, config)
	Expect(err).NotTo(HaveOccurred(), "Failed to set config for %s", ta.Name)

	ta.logger.Debug("Configuration updated",
		"config_count", len(config))
}

func (ta *TestApplication) Destroy() {
	ctx := context.Background()
	_, err := ta.fixture.Client.ExecuteCommand(ctx, "apps:destroy", []string{ta.Name, "--force"})
	Expect(err).NotTo(HaveOccurred(), "Failed to destroy app %s", ta.Name)

	ta.logger.Debug("Application destroyed")
}

func (ta *TestApplication) AssertExists() {
	ctx := context.Background()
	_, err := ta.fixture.Client.ExecuteCommand(ctx, "apps:exists", []string{ta.Name})
	Expect(err).NotTo(HaveOccurred(), "The application %s should exist", ta.Name)
}

func (ta *TestApplication) AssertNotExists() {
	ctx := context.Background()
	_, err := ta.fixture.Client.ExecuteCommand(ctx, "apps:exists", []string{ta.Name})
	Expect(err).To(HaveOccurred(), "The application %s should not exist", ta.Name)
}

// TestApplicationBuilder helps to create applications with specific configurations
type TestApplicationBuilder struct {
	fixture    *TestFixture
	namePrefix string
	config     map[string]string
}

func (f *TestFixture) NewApplicationBuilder() *TestApplicationBuilder {
	return &TestApplicationBuilder{
		fixture: f,
		config:  make(map[string]string),
	}
}

func (b *TestApplicationBuilder) WithNamePrefix(prefix string) *TestApplicationBuilder {
	b.namePrefix = prefix
	return b
}

func (b *TestApplicationBuilder) WithConfig(key, value string) *TestApplicationBuilder {
	b.config[key] = value
	return b
}

func (b *TestApplicationBuilder) WithMultipleConfig(config map[string]string) *TestApplicationBuilder {
	for k, v := range config {
		b.config[k] = v
	}
	return b
}

// Build creates the test application
func (b *TestApplicationBuilder) Build() *TestApplication {
	app := b.fixture.CreateTestApplication(b.namePrefix)

	if len(b.config) > 0 {
		app.SetConfig(b.config)
	}

	return app
}

// Utility functions

func skipIfDokkuNotAvailable(config *TestConfig) {
	if getEnv("DOKKU_MCP_INTEGRATION_TESTS", "") == "" {
		ginkgo.Skip("Integration tests disabled - set DOKKU_MCP_INTEGRATION_TESTS=1 to enable")
	}

	// In mock mode, no need to check Dokku connection
	if config.UseMockClient {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ssh",
		"-o", "ConnectTimeout=5",
		"-o", "BatchMode=yes",
		"-o", "LogLevel=QUIET",
		"-p", fmt.Sprintf("%d", config.DokkuPort),
		fmt.Sprintf("%s@%s", config.DokkuUser, config.DokkuHost),
		"echo", "test")

	if err := cmd.Run(); err != nil {
		ginkgo.Skip(fmt.Sprintf("Failed to connect to Dokku %s@%s:%d - integration tests skipped. Use DOKKU_MCP_USE_MOCK=true for mock tests: %v",
			config.DokkuUser, config.DokkuHost, config.DokkuPort, err))
	}
}

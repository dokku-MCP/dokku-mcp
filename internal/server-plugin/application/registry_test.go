//go:build !integration

package plugins_test

import (
	"context"
	"io"
	"log/slog"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"

	plugins "github.com/alex-galey/dokku-mcp/internal/server-plugin/application"
	"github.com/alex-galey/dokku-mcp/internal/server-plugin/domain"
	"github.com/alex-galey/dokku-mcp/pkg/config"
	"github.com/mark3labs/mcp-go/server"
)

// createTestLogger creates a quiet logger for testing that discards output
func createTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelError, // Only show errors and above during tests
	}))
}

// MockServerPluginDiscoveryService is a mock implementation of ServerPluginDiscoveryService for testing
type MockServerPluginDiscoveryService struct {
	getEnabledServerPluginsFunc func(ctx context.Context) ([]string, error)
	callCount                   map[string]int
}

func NewMockServerPluginDiscoveryService() *MockServerPluginDiscoveryService {
	return &MockServerPluginDiscoveryService{
		callCount: make(map[string]int),
	}
}

func (m *MockServerPluginDiscoveryService) GetEnabledDokkuPlugins(ctx context.Context) ([]string, error) {
	m.callCount["GetEnabledDokkuPlugins"]++
	if m.getEnabledServerPluginsFunc != nil {
		return m.getEnabledServerPluginsFunc(ctx)
	}
	return []string{}, nil
}

func (m *MockServerPluginDiscoveryService) GetCallCount(method string) int {
	return m.callCount[method]
}

// MockServerPlugin is a mock implementation of ServerPlugin for testing
type MockServerPlugin struct {
	id          string
	name        string
	description string
	version     string
	pluginName  string
	callCount   map[string]int
}

func NewMockServerPlugin(name, pluginName string) *MockServerPlugin {
	return &MockServerPlugin{
		id:          name,
		name:        name,
		description: "Mock plugin for testing",
		version:     "1.0.0",
		pluginName:  pluginName,
		callCount:   make(map[string]int),
	}
}

func (m *MockServerPlugin) ID() string {
	return m.id
}

func (m *MockServerPlugin) Name() string {
	return m.name
}

func (m *MockServerPlugin) Description() string {
	return m.description
}

func (m *MockServerPlugin) Version() string {
	return m.version
}

func (m *MockServerPlugin) DokkuPluginName() string {
	return m.pluginName
}

func (m *MockServerPlugin) GetCallCount(method string) int {
	return m.callCount[method]
}

var _ = Describe("DynamicServerPluginRegistry", func() {
	var (
		registry         *plugins.DynamicServerPluginRegistry
		mockDiscovery    *MockServerPluginDiscoveryService
		mockServerPlugin *MockServerPlugin
		logger           *slog.Logger
		srvConfig        *config.ServerConfig
	)

	BeforeEach(func() {
		mockDiscovery = NewMockServerPluginDiscoveryService()
		mockServerPlugin = NewMockServerPlugin("test", "")
		logger = createTestLogger()
		srvConfig = config.DefaultConfig()
	})

	Describe("Basic Functionality", func() {
		BeforeEach(func() {
			// Set up mock expectations
			mockDiscovery.getEnabledServerPluginsFunc = func(ctx context.Context) ([]string, error) {
				return []string{"some-plugin"}, nil
			}

			// Create registry with correct arguments
			pluginRegistry := plugins.NewServerPluginRegistry()
			params := plugins.DynamicServerPluginRegistryParams{
				PluginRegistry:  pluginRegistry,
				PluginDiscovery: mockDiscovery,
				Logger:          logger,
				SrvConfig:       srvConfig,
				ServerPlugins:   []domain.ServerPlugin{mockServerPlugin},
			}
			registry = plugins.NewDynamicServerPluginRegistry(params)
		})

		Context("when syncing plugins", func() {
			It("should activate core plugins (empty plugin name)", func() {
				err := registry.SyncServerPlugins(context.Background())
				Expect(err).NotTo(HaveOccurred())

				Expect(registry.IsServerPluginActive("test")).To(BeTrue(), "Core plugin should be active")

				activeServerPlugins := registry.GetActiveServerPlugins()
				Expect(activeServerPlugins).To(HaveLen(1), "Should have exactly one active plugin")
				Expect(activeServerPlugins[0].Name()).To(Equal("test"))

				Expect(mockDiscovery.GetCallCount("GetEnabledDokkuPlugins")).To(Equal(1))
			})
		})

		Context("when checking plugin status", func() {
			BeforeEach(func() {
				err := registry.SyncServerPlugins(context.Background())
				Expect(err).NotTo(HaveOccurred())
			})

			It("should correctly report active plugins", func() {
				Expect(registry.IsServerPluginActive("test")).To(BeTrue())
				Expect(registry.IsServerPluginActive("nonexistent")).To(BeFalse())
			})

			It("should return list of active plugins", func() {
				activeServerPlugins := registry.GetActiveServerPlugins()
				Expect(activeServerPlugins).To(HaveLen(1))
				Expect(activeServerPlugins[0].Name()).To(Equal("test"))
			})
		})
	})

	Describe("ServerPlugin-based ServerPlugin Activation", func() {
		var postgresServerPlugin *MockServerPlugin

		BeforeEach(func() {
			postgresServerPlugin = NewMockServerPlugin("postgres", "postgres")

			pluginRegistry := plugins.NewServerPluginRegistry()
			params := plugins.DynamicServerPluginRegistryParams{
				PluginRegistry:  pluginRegistry,
				PluginDiscovery: mockDiscovery,
				Logger:          logger,
				SrvConfig:       srvConfig,
				ServerPlugins:   []domain.ServerPlugin{postgresServerPlugin},
			}
			registry = plugins.NewDynamicServerPluginRegistry(params)
		})

		Context("when plugin is enabled", func() {
			BeforeEach(func() {
				mockDiscovery.getEnabledServerPluginsFunc = func(ctx context.Context) ([]string, error) {
					return []string{"postgres"}, nil
				}
			})

			It("should activate the corresponding plugin", func() {
				err := registry.SyncServerPlugins(context.Background())
				Expect(err).NotTo(HaveOccurred())

				Expect(registry.IsServerPluginActive("postgres")).To(BeTrue())
			})
		})

		Context("when plugin is disabled", func() {
			BeforeEach(func() {
				// First enable the plugin
				mockDiscovery.getEnabledServerPluginsFunc = func(ctx context.Context) ([]string, error) {
					return []string{"postgres"}, nil
				}
				err := registry.SyncServerPlugins(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(registry.IsServerPluginActive("postgres")).To(BeTrue())

				// Then disable it
				mockDiscovery.getEnabledServerPluginsFunc = func(ctx context.Context) ([]string, error) {
					return []string{}, nil
				}
			})

			It("should deactivate the corresponding plugin", func() {
				err := registry.SyncServerPlugins(context.Background())
				Expect(err).NotTo(HaveOccurred())

				Expect(registry.IsServerPluginActive("postgres")).To(BeFalse())
			})
		})
	})

	Describe("Fx Integration", func() {
		Context("when using Fx lifecycle", func() {
			It("should integrate properly with dependency injection", func() {
				var testRegistry *plugins.DynamicServerPluginRegistry

				mockDiscovery := NewMockServerPluginDiscoveryService()
				mockServerPlugin := NewMockServerPlugin("test", "")

				mockDiscovery.getEnabledServerPluginsFunc = func(ctx context.Context) ([]string, error) {
					return []string{"some-plugin"}, nil
				}

				app := fx.New(
					fx.Provide(
						func() domain.ServerPluginDiscoveryService { return mockDiscovery },
						func() *slog.Logger { return createTestLogger() },
						func() *server.MCPServer {
							return server.NewMCPServer("test", "1.0.0")
						},
						func() []domain.ServerPlugin { return []domain.ServerPlugin{mockServerPlugin} },
						plugins.NewServerPluginRegistry,
						func() *config.ServerConfig { return config.DefaultConfig() },
						plugins.NewDynamicServerPluginRegistry,
					),
					fx.Populate(&testRegistry),
					fx.NopLogger, // Suppress Fx logs during testing
				)

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				err := app.Start(ctx)
				Expect(err).NotTo(HaveOccurred())

				Expect(testRegistry).NotTo(BeNil())

				err = app.Stop(ctx)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})

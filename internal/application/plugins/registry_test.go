package plugins_test

import (
	"context"
	"log/slog"
	"time"

	"github.com/alex-galey/dokku-mcp/internal/application/plugins"
	"github.com/alex-galey/dokku-mcp/internal/domain"
	"github.com/mark3labs/mcp-go/server"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/fx"
)

// MockPluginDiscoveryService is a mock implementation of PluginDiscoveryService for testing
type MockPluginDiscoveryService struct {
	getEnabledPluginsFunc func(ctx context.Context) ([]string, error)
	callCount             map[string]int
}

func NewMockPluginDiscoveryService() *MockPluginDiscoveryService {
	return &MockPluginDiscoveryService{
		callCount: make(map[string]int),
	}
}

func (m *MockPluginDiscoveryService) GetEnabledPlugins(ctx context.Context) ([]string, error) {
	m.callCount["GetEnabledPlugins"]++
	if m.getEnabledPluginsFunc != nil {
		return m.getEnabledPluginsFunc(ctx)
	}
	return []string{}, nil
}

func (m *MockPluginDiscoveryService) GetCallCount(method string) int {
	return m.callCount[method]
}

// MockFeaturePlugin is a mock implementation of FeaturePlugin for testing
type MockFeaturePlugin struct {
	name         string
	pluginName   string
	registerFunc func(*server.MCPServer) error
	callCount    map[string]int
}

func NewMockFeaturePlugin(name, pluginName string) *MockFeaturePlugin {
	return &MockFeaturePlugin{
		name:       name,
		pluginName: pluginName,
		callCount:  make(map[string]int),
	}
}

func (m *MockFeaturePlugin) Name() string {
	return m.name
}

func (m *MockFeaturePlugin) DokkuPluginName() string {
	return m.pluginName
}

func (m *MockFeaturePlugin) Register(mcpServer *server.MCPServer) error {
	m.callCount["Register"]++
	if m.registerFunc != nil {
		return m.registerFunc(mcpServer)
	}
	return nil
}

func (m *MockFeaturePlugin) Deregister(mcpServer *server.MCPServer) error {
	m.callCount["Deregister"]++
	return nil
}

func (m *MockFeaturePlugin) GetCallCount(method string) int {
	return m.callCount[method]
}

var _ = Describe("DynamicPluginRegistry", func() {
	var (
		registry      *plugins.DynamicPluginRegistry
		mockDiscovery *MockPluginDiscoveryService
		mockPlugin    *MockFeaturePlugin
		logger        *slog.Logger
		mcpServer     *server.MCPServer
	)

	BeforeEach(func() {
		mockDiscovery = NewMockPluginDiscoveryService()
		mockPlugin = NewMockFeaturePlugin("test", "")
		logger = slog.Default()
		mcpServer = server.NewMCPServer("test", "1.0.0")
	})

	Describe("Basic Functionality", func() {
		BeforeEach(func() {
			// Set up mock expectations
			mockDiscovery.getEnabledPluginsFunc = func(ctx context.Context) ([]string, error) {
				return []string{"some-plugin"}, nil
			}

			// Create registry directly
			registry = plugins.NewDynamicPluginRegistry(
				mcpServer,
				mockDiscovery,
				logger,
				[]domain.FeaturePlugin{mockPlugin},
			)
		})

		Context("when syncing plugins", func() {
			It("should activate core plugins (empty plugin name)", func() {
				err := registry.SyncPlugins(context.Background())
				Expect(err).NotTo(HaveOccurred())

				Expect(registry.IsPluginActive("test")).To(BeTrue(), "Core plugin should be active")

				activePlugins := registry.GetActivePlugins()
				Expect(activePlugins).To(HaveLen(1), "Should have exactly one active plugin")
				Expect(activePlugins[0].Name()).To(Equal("test"))

				Expect(mockDiscovery.GetCallCount("GetEnabledPlugins")).To(Equal(1))
				Expect(mockPlugin.GetCallCount("Register")).To(Equal(1))
			})
		})

		Context("when checking plugin status", func() {
			BeforeEach(func() {
				err := registry.SyncPlugins(context.Background())
				Expect(err).NotTo(HaveOccurred())
			})

			It("should correctly report active plugins", func() {
				Expect(registry.IsPluginActive("test")).To(BeTrue())
				Expect(registry.IsPluginActive("nonexistent")).To(BeFalse())
			})

			It("should return list of active plugins", func() {
				activePlugins := registry.GetActivePlugins()
				Expect(activePlugins).To(HaveLen(1))
				Expect(activePlugins[0].Name()).To(Equal("test"))
			})
		})
	})

	Describe("Plugin-based Plugin Activation", func() {
		var postgresPlugin *MockFeaturePlugin

		BeforeEach(func() {
			postgresPlugin = NewMockFeaturePlugin("postgres", "postgres")

			registry = plugins.NewDynamicPluginRegistry(
				mcpServer,
				mockDiscovery,
				logger,
				[]domain.FeaturePlugin{postgresPlugin},
			)
		})

		Context("when plugin is enabled", func() {
			BeforeEach(func() {
				mockDiscovery.getEnabledPluginsFunc = func(ctx context.Context) ([]string, error) {
					return []string{"postgres"}, nil
				}
			})

			It("should activate the corresponding plugin", func() {
				err := registry.SyncPlugins(context.Background())
				Expect(err).NotTo(HaveOccurred())

				Expect(registry.IsPluginActive("postgres")).To(BeTrue())
				Expect(postgresPlugin.GetCallCount("Register")).To(Equal(1))
			})
		})

		Context("when plugin is disabled", func() {
			BeforeEach(func() {
				// First enable the plugin
				mockDiscovery.getEnabledPluginsFunc = func(ctx context.Context) ([]string, error) {
					return []string{"postgres"}, nil
				}
				err := registry.SyncPlugins(context.Background())
				Expect(err).NotTo(HaveOccurred())
				Expect(registry.IsPluginActive("postgres")).To(BeTrue())

				// Then disable it
				mockDiscovery.getEnabledPluginsFunc = func(ctx context.Context) ([]string, error) {
					return []string{}, nil
				}
			})

			It("should deactivate the corresponding plugin", func() {
				err := registry.SyncPlugins(context.Background())
				Expect(err).NotTo(HaveOccurred())

				Expect(registry.IsPluginActive("postgres")).To(BeFalse())
				Expect(postgresPlugin.GetCallCount("Register")).To(Equal(1))
				Expect(postgresPlugin.GetCallCount("Deregister")).To(Equal(1))
			})
		})
	})

	Describe("Fx Integration", func() {
		Context("when using Fx lifecycle", func() {
			It("should integrate properly with dependency injection", func() {
				var testRegistry *plugins.DynamicPluginRegistry

				mockDiscovery := NewMockPluginDiscoveryService()
				mockPlugin := NewMockFeaturePlugin("test", "")

				mockDiscovery.getEnabledPluginsFunc = func(ctx context.Context) ([]string, error) {
					return []string{"some-plugin"}, nil
				}

				app := fx.New(
					fx.Provide(
						func() domain.PluginDiscoveryService { return mockDiscovery },
						func() *slog.Logger { return slog.Default() },
						func() *server.MCPServer {
							return server.NewMCPServer("test", "1.0.0")
						},
						func() []domain.FeaturePlugin { return []domain.FeaturePlugin{mockPlugin} },
						plugins.NewDynamicPluginRegistry,
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

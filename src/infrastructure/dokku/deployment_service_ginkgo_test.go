//go:build integration

package dokku

import (
	"context"
	"log/slog"
	"time"

	"github.com/alex-galey/dokku-mcp/src/domain/application"
	dokkutesting "github.com/alex-galey/dokku-mcp/testing/dokku"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Adapters to implement testing package interfaces

// dokkuClientFactory implements DokkuClientFactory
type dokkuClientFactory struct{}

func (f *dokkuClientFactory) CreateClient(config *dokkutesting.ClientConfig, logger *slog.Logger) dokkutesting.DokkuClient {
	clientConfig := &ClientConfig{
		DokkuHost:       config.DokkuHost,
		DokkuPort:       config.DokkuPort,
		DokkuUser:       config.DokkuUser,
		SSHKeyPath:      config.SSHKeyPath,
		CommandTimeout:  config.CommandTimeout,
		AllowedCommands: config.AllowedCommands,
	}
	return NewDokkuClient(clientConfig, logger)
}

// deploymentServiceFactory implements DeploymentServiceFactory
type deploymentServiceFactory struct{}

func (f *deploymentServiceFactory) CreateService(client dokkutesting.DokkuClient, logger *slog.Logger) application.DeploymentService {
	// The DokkuClient adapter to concrete type is not necessary because DokkuClient already implements the interface
	// But we need to create a type that implements our local interface
	dokkuClient, ok := client.(DokkuClient)
	if !ok {
		panic("The provided client cannot be converted to DokkuClient")
	}
	return NewDeploymentService(dokkuClient, logger)
}

var _ = Describe("DeploymentService - Integration Tests", func() {
	var (
		fixture *dokkutesting.TestFixture
		ctx     context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()

		// Default configuration automatically uses mocks
		config := dokkutesting.LoadTestConfig()

		// Only configure real factories if not in mock mode
		if !config.UseMockClient {
			config.SetFactories(&dokkuClientFactory{}, &deploymentServiceFactory{})
		}

		fixture = dokkutesting.NewTestFixtureWithConfig(config)
		fixture.Logger.Info("Integration test started", "mock_mode", config.UseMockClient)
	})

	Describe("Basic Dokku client operations", func() {
		Context("Application listing", func() {
			It("should return a list without error", func() {
				apps, err := fixture.Client.GetApplications(ctx)

				Expect(err).NotTo(HaveOccurred())
				Expect(apps).NotTo(BeNil())

				fixture.Logger.Info("Applications listed successfully",
					"number_apps", len(apps))
			})
		})

		Context("Checking non-existent applications", func() {
			It("should return an error", func() {
				appName := fixture.Config.TestAppPrefix + "-nonexistent-123456789"

				_, err := fixture.Client.ExecuteCommand(ctx, "apps:exists", []string{appName})

				Expect(err).To(HaveOccurred(), "A non-existent application should return an error")
			})
		})
	})

	DescribeTable("GetHistory for different application types",
		func(setupApp bool, appNameSuffix string) {
			var appName string
			var testApp *dokkutesting.TestApplication

			if setupApp {
				testApp = fixture.CreateTestApplication("history-test")
				appName = testApp.Name
			} else {
				appName = fixture.Config.TestAppPrefix + "-nonexistent-" + appNameSuffix
			}

			deployments, err := fixture.DeploymentService.GetHistory(ctx, appName)

			Expect(err).NotTo(HaveOccurred(), "GetHistory should not return an error")
			Expect(deployments).NotTo(BeNil())

			if setupApp {
				// For an existing application, check deployment properties
				for i, deployment := range deployments {
					Expect(deployment.AppName).To(Equal(appName),
						"Deployment %d should have the correct app name", i)
					Expect(deployment.ID).NotTo(BeEmpty(),
						"Deployment %d should have a non-empty ID", i)
					Expect(deployment.CreatedAt).NotTo(BeZero(),
						"Deployment %d should have a creation date", i)
				}

				fixture.Logger.Info("History retrieved",
					"number_deployments", len(deployments))
			} else {
				// For a non-existent application, empty list
				Expect(deployments).To(BeEmpty(), "A non-existent application should return an empty list")
			}
		},
		Entry("with a non-existent application", false, dokkutesting.GenerateTestSuffix()),
		Entry("with an existing application", true, ""),
	)

	Describe("Application information", func() {
		var testApp *dokkutesting.TestApplication

		BeforeEach(func() {
			testApp = fixture.CreateTestApplication("info-test")
		})

		It("should retrieve basic information", func() {
			info := testApp.GetInfo()

			Expect(info).NotTo(BeNil())
			Expect(info).NotTo(BeEmpty())
		})

		It("should handle application configuration", func() {
			By("Retrieving initial configuration")
			initialConfig := testApp.GetConfig()
			Expect(initialConfig).NotTo(BeNil())

			By("Setting new configuration")
			newConfig := map[string]string{
				"TEST_VAR": "test_value",
				"ENV":      "test",
			}
			testApp.SetConfig(newConfig)

			By("Verifying that configuration was applied")
			updatedConfig := testApp.GetConfig()
			Expect(updatedConfig).To(HaveKey("TEST_VAR"))
			Expect(updatedConfig["TEST_VAR"]).To(Equal("test_value"))
		})
	})

	Describe("Event parsing", func() {
		BeforeEach(func() {
			if fixture.Config.UseMockClient {
				Skip("This test requires a real Dokku server (tests an internal method)")
			}
			if !fixture.Config.EventsEnabled {
				Skip("Dokku events are not available")
			}
		})

		It("should retrieve and parse events", func() {
			ctx := context.Background()
			output, err := fixture.Client.ExecuteCommand(ctx, "events", []string{})
			Expect(err).NotTo(HaveOccurred())

			eventsOutput := string(output)
			Expect(eventsOutput).NotTo(BeNil())
			fixture.Logger.Info("Events retrieved",
				"events_size", len(eventsOutput))

			if len(eventsOutput) > 0 {
				service := fixture.DeploymentService.(*deploymentService)
				deployments, err := service.parseEventsOutput(eventsOutput, "test-parsing")

				Expect(err).NotTo(HaveOccurred())
				Expect(deployments).NotTo(BeNil())
			}
		})
	})

	Describe("git:report fallback mechanism", func() {
		BeforeEach(func() {
			if fixture.Config.UseMockClient {
				Skip("This test requires a real Dokku server (tests an internal method)")
			}
			if !fixture.Config.GitReportEnabled {
				Skip("git:report Dokku is not available")
			}
		})

		It("should use fallback correctly", func() {
			testApp := fixture.CreateTestApplication("git-report-test")

			service := fixture.DeploymentService.(*deploymentService)
			deployments, err := service.getCurrentDeploymentAsFallback(ctx, testApp.Name)

			Expect(err).NotTo(HaveOccurred())
			Expect(deployments).NotTo(BeNil())

			if len(deployments) > 0 {
				deployment := deployments[0]
				Expect(deployment.AppName).To(Equal(testApp.Name))
				Expect(deployment.ID).NotTo(BeEmpty())
				Expect(deployment.Status).To(Equal(application.DeploymentStatusSucceeded))
			}
		})
	})

	Describe("Builder pattern for applications", func() {
		It("should create a basic application", func() {
			app := fixture.NewApplicationBuilder().
				WithNamePrefix("builder-test").
				Build()

			Expect(app).NotTo(BeNil())
			Expect(app.Name).To(ContainSubstring("builder-test"))
			app.AssertExists()
		})

		It("should create an application with configuration", func() {
			config := map[string]string{
				"TEST_VAR": "test_value",
				"ENV":      "test",
			}

			app := fixture.NewApplicationBuilder().
				WithNamePrefix("config-test").
				WithMultipleConfig(config).
				Build()

			Expect(app).NotTo(BeNil())

			// Verify that configuration was applied
			appConfig := app.GetConfig()
			Expect(appConfig).To(HaveKey("TEST_VAR"))
			Expect(appConfig["TEST_VAR"]).To(Equal("test_value"))
		})
	})

	Describe("Multiple application management", func() {
		BeforeEach(func() {
			if fixture.Config.SkipSlowTests {
				Skip("Slow tests disabled")
			}
		})

		It("should create and manage multiple applications", func() {
			const appCount = 3
			apps := fixture.CreateMultipleTestApplications("multi-test", appCount)

			Expect(apps).To(HaveLen(appCount))

			for i, app := range apps {
				Expect(app).NotTo(BeNil(), "Application %d should not be nil", i)
				Expect(app.Name).To(ContainSubstring("multi-test"))
				app.AssertExists()
			}

			fixture.Logger.Info("Multiple applications created successfully",
				"count", len(apps))
		})
	})

	DescribeTable("Error handling",
		func(command string, args []string, shouldError bool, description string) {
			_, err := fixture.Client.ExecuteCommand(ctx, command, args)

			if shouldError {
				Expect(err).To(HaveOccurred(), description)
			} else {
				Expect(err).NotTo(HaveOccurred(), description)
			}
		},
		Entry("creation with invalid name", "apps:create", []string{"invalid/app/name"}, true,
			"Creating an app with an invalid name should fail"),
		Entry("deleting non-existent app", "apps:destroy",
			[]string{dokkutesting.GenerateTestSuffix(), "--force"}, true,
			"Deleting a non-existent app should fail"),
	)

	Describe("Performance", func() {
		var testApp *dokkutesting.TestApplication

		BeforeEach(func() {
			if CurrentSpecReport().IsSerial {
				Skip("Skip benchmarks in short mode")
			}
			testApp = fixture.CreateTestApplication("bench")
			time.Sleep(2 * time.Second)
		})

		It("should execute GetHistory in reasonable time", func() {
			const iterations = 10
			start := time.Now()

			for i := 0; i < iterations; i++ {
				_, err := fixture.DeploymentService.GetHistory(ctx, testApp.Name)
				Expect(err).NotTo(HaveOccurred())
			}

			duration := time.Since(start)
			avgDuration := duration / iterations

			fixture.Logger.Info("Benchmark GetHistory completed",
				"total_duration", duration,
				"average_duration", avgDuration,
				"iterations", iterations)

			Expect(avgDuration).To(BeNumerically("<", 5*time.Second),
				"GetHistory should run in less than 5 seconds on average")
		})
	})

	Describe("Complete workflow", func() {
		BeforeEach(func() {
			if fixture.Config.SkipSlowTests {
				Skip("Slow tests disabled")
			}
		})

		It("should handle complete application lifecycle", func() {
			By("Verifying that a non-existent app returns an empty list")
			appName := fixture.Config.TestAppPrefix + "-workflow-" + dokkutesting.GenerateTestSuffix()
			deployments, err := fixture.DeploymentService.GetHistory(ctx, appName)
			Expect(err).NotTo(HaveOccurred())
			Expect(deployments).To(BeEmpty())

			By("Creating an application and checking its history")
			testApp := fixture.CreateTestApplication("workflow")

			time.Sleep(2 * time.Second)

			deployments = testApp.GetHistory()
			Expect(deployments).NotTo(BeNil())

			// Check if we have deployments
			for _, deployment := range deployments {
				Expect(deployment.AppName).To(Equal(testApp.Name))
				Expect(deployment.ID).NotTo(BeEmpty())
				Expect(deployment.CreatedAt).NotTo(BeZero())
			}
		})
	})
})

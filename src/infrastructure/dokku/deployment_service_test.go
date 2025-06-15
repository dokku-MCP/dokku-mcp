package dokku

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alex-galey/dokku-mcp/src/domain/application"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DeploymentService Unit Tests", func() {
	var (
		mockClient       *MockDokkuClient
		service          *deploymentService
		ctx              context.Context
		testAppName      string
		mockEventsOutput string
		gitReportOutput  string
	)

	BeforeEach(func() {
		mockClient = NewMockDokkuClient()

		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		service = &deploymentService{
			client: mockClient,
			logger: logger,
		}

		ctx = context.Background()
		testAppName = "test-app"

		// Données de test standardisées
		mockEventsOutput = `Jul  3 16:09:48 dokku.me dokku[127630]: INVOKED: pre-deploy( test-app )
Jul  3 16:10:24 dokku.me dokku[129451]: INVOKED: check-deploy( test-app 6274ced0d4be11af4490cd18abaf77cdd593f025133f403d984e80d86a39acec web 5000 10.0.16.80 )
Jul  3 16:10:46 dokku.me dokku[129851]: INVOKED: post-deploy( test-app )
Jul  3 16:15:02 dokku.me dokku[130397]: INVOKED: pre-deploy( test-app )
Jul  3 16:15:30 dokku.me dokku[130500]: INVOKED: check-deploy( test-app abcd1234567890abcdef1234567890abcdef1234 web 5000 10.0.16.81 )
Jul  3 16:15:45 dokku.me dokku[130600]: INVOKED: post-deploy( test-app )`

		gitReportOutput = "=====> test-app git information\nGit sha:                   abcd1234567890abcdef\nDeploy source:            git\n"
	})

	Describe("GetHistory", func() {
		Context("When the application exists", func() {
			BeforeEach(func() {
				mockClient.SetExecuteCommandResponse("apps:exists", []string{testAppName}, []byte(""), nil)
			})

			Context("With available events", func() {
				BeforeEach(func() {
					mockClient.SetExecuteCommandResponse("events", []string{}, []byte(mockEventsOutput), nil)
				})

				It("should retrieve and parse the deployment history", func() {
					deployments, err := service.GetHistory(ctx, testAppName)

					Expect(err).NotTo(HaveOccurred())
					Expect(deployments).NotTo(BeEmpty())

					// Check that deployments are sorted by descending date
					if len(deployments) > 1 {
						for i := 0; i < len(deployments)-1; i++ {
							Expect(deployments[i].CreatedAt.After(deployments[i+1].CreatedAt) ||
								deployments[i].CreatedAt.Equal(deployments[i+1].CreatedAt)).To(BeTrue(),
								"Deployments should be sorted by descending date")
						}
					}

					// Check that at least one deployment has an extracted SHA
					hasSHA := false
					for _, deployment := range deployments {
						Expect(deployment.AppName).To(Equal(testAppName))
						Expect(deployment.ID).NotTo(BeEmpty())

						if deployment.GitRef != "unknown" && len(deployment.GitRef) >= 8 {
							hasSHA = true
						}
					}
					Expect(hasSHA).To(BeTrue(), "At least one deployment should have an extracted SHA")
				})
			})

			Context("With fallback git:report", func() {
				BeforeEach(func() {
					mockClient.SetExecuteCommandResponse("events", []string{}, []byte(""), errors.New("events not available"))
					mockClient.SetExecuteCommandResponse("git:report", []string{testAppName}, []byte(gitReportOutput), nil)
				})

				It("should use git:report as fallback", func() {
					deployments, err := service.GetHistory(ctx, testAppName)

					Expect(err).NotTo(HaveOccurred())
					Expect(deployments).To(HaveLen(1))
					Expect(deployments[0].AppName).To(Equal(testAppName))
					Expect(deployments[0].GitRef).To(Equal("abcd1234"))
					Expect(deployments[0].Status).To(Equal(application.DeploymentStatusSucceeded))
				})
			})
		})

		Context("When the application does not exist", func() {
			It("should return an empty list", func() {
				mockClient.SetExecuteCommandResponse("apps:exists", []string{testAppName}, nil, errors.New("app does not exist"))

				deployments, err := service.GetHistory(ctx, testAppName)

				Expect(err).NotTo(HaveOccurred())
				Expect(deployments).NotTo(BeNil())
				Expect(deployments).To(BeEmpty())
			})
		})
	})

	Describe("parseEventsOutput", func() {
		Context("With valid events", func() {
			It("should parse the events correctly", func() {
				deployments, err := service.parseEventsOutput(mockEventsOutput, testAppName)

				Expect(err).NotTo(HaveOccurred())
				Expect(deployments).NotTo(BeEmpty())

				// Check that only events for test-app are returned
				for _, deployment := range deployments {
					Expect(deployment.AppName).To(Equal(testAppName))
				}

				// Check that at least one deployment has the extracted SHA
				foundSHA := false
				for _, deployment := range deployments {
					if strings.HasPrefix(deployment.GitRef, "6274ced0") {
						foundSHA = true
						break
					}
				}
				Expect(foundSHA).To(BeTrue(), "The SHA '6274ced0' should be extracted from the events")
			})
		})

		Context("With events from other applications", func() {
			It("should filter events by application name", func() {
				mixedEvents := mockEventsOutput + `
Jul  4 08:15:30 dokku.me dokku[130500]: INVOKED: check-deploy( other-app abcd1234567890abcdef1234567890abcdef1234 web 5000 10.0.16.81 )
Jul  5 09:20:15 dokku.me dokku[131000]: INVOKED: pre-deploy( another-app )`

				deployments, err := service.parseEventsOutput(mixedEvents, testAppName)

				Expect(err).NotTo(HaveOccurred())
				for _, deployment := range deployments {
					Expect(deployment.AppName).To(Equal(testAppName),
						"Only events for %s should be returned", testAppName)
				}
			})
		})
	})

	DescribeTable("parseEventTimestamp",
		func(timestampStr string, expectError bool) {
			timestamp, err := service.parseEventTimestamp(timestampStr)

			if expectError {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(timestamp).NotTo(BeZero())

				// Check specific parsing for the valid timestamp
				if timestampStr == "Jul  3 16:10:24" {
					Expect(timestamp.Month()).To(Equal(time.July))
					Expect(timestamp.Day()).To(Equal(3))
					Expect(timestamp.Hour()).To(Equal(16))
					Expect(timestamp.Minute()).To(Equal(10))
					Expect(timestamp.Second()).To(Equal(24))
				}
			}
		},
		Entry("with a valid timestamp", "Jul  3 16:10:24", false),
		Entry("with an invalid timestamp", "invalid-timestamp", true),
		Entry("with an empty timestamp", "", true),
	)

	Describe("getCurrentDeploymentAsFallback", func() {
		Context("Quand git:report retourne des informations", func() {
			BeforeEach(func() {
				mockClient.SetExecuteCommandResponse("git:report", []string{testAppName}, []byte(gitReportOutput), nil)
			})

			It("should create a deployment from git:report", func() {
				deployments, err := service.getCurrentDeploymentAsFallback(ctx, testAppName)

				Expect(err).NotTo(HaveOccurred())
				Expect(deployments).To(HaveLen(1))

				deployment := deployments[0]
				Expect(deployment.AppName).To(Equal(testAppName))
				Expect(deployment.GitRef).To(Equal("abcd1234"))
				Expect(deployment.Status).To(Equal(application.DeploymentStatusSucceeded))
				Expect(deployment.CreatedAt).NotTo(BeZero())
			})
		})

		Context("When git:report fails", func() {
			BeforeEach(func() {
				mockClient.SetExecuteCommandResponse("git:report", []string{testAppName}, nil, errors.New("git report failed"))
			})

			It("should return an empty list", func() {
				deployments, err := service.getCurrentDeploymentAsFallback(ctx, testAppName)

				Expect(err).NotTo(HaveOccurred())
				Expect(deployments).To(BeEmpty())
			})
		})
	})
})

// MockDokkuClient est un mock simple pour les tests unitaires
type MockDokkuClient struct {
	responses map[string]MockResponse
}

type MockResponse struct {
	Data  []byte
	Error error
}

func NewMockDokkuClient() *MockDokkuClient {
	return &MockDokkuClient{
		responses: make(map[string]MockResponse),
	}
}

// SetExecuteCommandResponse configure la réponse pour une commande donnée
func (m *MockDokkuClient) SetExecuteCommandResponse(command string, args []string, data []byte, err error) {
	key := command + "|" + strings.Join(args, ",")
	m.responses[key] = MockResponse{Data: data, Error: err}
}

func (m *MockDokkuClient) ExecuteCommand(ctx context.Context, command string, args []string) ([]byte, error) {
	key := command + "|" + strings.Join(args, ",")
	if response, exists := m.responses[key]; exists {
		return response.Data, response.Error
	}
	return nil, errors.New("mock response not configured")
}

func (m *MockDokkuClient) GetApplications(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *MockDokkuClient) GetApplicationInfo(ctx context.Context, appName string) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

func (m *MockDokkuClient) GetApplicationConfig(ctx context.Context, appName string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (m *MockDokkuClient) SetApplicationConfig(ctx context.Context, appName string, config map[string]string) error {
	return nil
}

func (m *MockDokkuClient) ScaleApplication(ctx context.Context, appName string, processType string, count int) error {
	return nil
}

func (m *MockDokkuClient) GetApplicationLogs(ctx context.Context, appName string, lines int) (string, error) {
	return "", nil
}

func TestNewDeploymentService(t *testing.T) {
	client := NewMockDokkuClient()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	service := NewDeploymentService(client, logger)
	if service == nil {
		t.Fatal("Deployment service should not be nil")
	}
}

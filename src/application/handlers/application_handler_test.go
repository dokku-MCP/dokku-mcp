package handlers_test

import (
	"context"
	"errors"
	"log/slog"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/alex-galey/dokku-mcp/src/application/handlers"
	"github.com/alex-galey/dokku-mcp/src/domain/application"
)

type MockRepository struct {
	applications map[string]*application.Application
	saveError    error
	getError     error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		applications: make(map[string]*application.Application),
	}
}

func (m *MockRepository) GetAll(ctx context.Context) ([]*application.Application, error) {
	if m.getError != nil {
		return nil, m.getError
	}

	apps := make([]*application.Application, 0, len(m.applications))
	for _, app := range m.applications {
		apps = append(apps, app)
	}
	return apps, nil
}

func (m *MockRepository) GetByName(ctx context.Context, name string) (*application.Application, error) {
	if m.getError != nil {
		return nil, m.getError
	}

	app, exists := m.applications[name]
	if !exists {
		return nil, application.ErrApplicationNotFound
	}
	return app, nil
}

func (m *MockRepository) Save(ctx context.Context, app *application.Application) error {
	if m.saveError != nil {
		return m.saveError
	}

	m.applications[app.Name()] = app
	return nil
}

func (m *MockRepository) Delete(ctx context.Context, name string) error {
	delete(m.applications, name)
	return nil
}

func (m *MockRepository) Exists(ctx context.Context, name string) (bool, error) {
	_, exists := m.applications[name]
	return exists, nil
}

func (m *MockRepository) List(ctx context.Context, offset, limit int) ([]*application.Application, int, error) {
	apps, err := m.GetAll(ctx)
	return apps, len(apps), err
}

func (m *MockRepository) GetByState(ctx context.Context, state application.ApplicationState) ([]*application.Application, error) {
	apps, err := m.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	filtered := make([]*application.Application, 0)
	for _, app := range apps {
		if app.State() == state {
			filtered = append(filtered, app)
		}
	}
	return filtered, nil
}

// Helper methods to configure the mock
func (m *MockRepository) SetSaveError(err error) {
	m.saveError = err
}

func (m *MockRepository) SetGetError(err error) {
	m.getError = err
}

func (m *MockRepository) AddApplication(app *application.Application) {
	m.applications[app.Name()] = app
}

var _ = Describe("ApplicationHandler", func() {
	var (
		handler  *handlers.ApplicationHandler
		mockRepo *MockRepository
		logger   *slog.Logger
	)

	BeforeEach(func() {
		mockRepo = NewMockRepository()
		logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
		handler = handlers.NewApplicationHandler(mockRepo, logger)
	})

	Describe("Creating the handler", func() {
		It("should create a handler with the provided dependencies", func() {
			Expect(handler).NotTo(BeNil())
		})

		It("should accept a nil repository and a nil logger without panicking", func() {
			Expect(func() {
				handlers.NewApplicationHandler(nil, nil)
			}).NotTo(Panic())
		})
	})

	Describe("Resource and tool registration", func() {
		Context("with existing applications", func() {
			BeforeEach(func() {
				app1, _ := application.NewApplication("app-test-1")
				app2, _ := application.NewApplication("app-test-2")
				mockRepo.AddApplication(app1)
				mockRepo.AddApplication(app2)
			})
		})
	})
})

var _ = Describe("MockRepository", func() {
	var mockRepo *MockRepository

	BeforeEach(func() {
		mockRepo = NewMockRepository()
	})

	Describe("Base operations", func() {
		It("should be empty", func() {
			apps, err := mockRepo.GetAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(apps).To(BeEmpty())
		})

		It("should save and retrieve applications", func() {
			app, err := application.NewApplication("test-app")
			Expect(err).NotTo(HaveOccurred())

			err = mockRepo.Save(context.Background(), app)
			Expect(err).NotTo(HaveOccurred())

			retrievedApp, err := mockRepo.GetByName(context.Background(), "test-app")
			Expect(err).NotTo(HaveOccurred())
			Expect(retrievedApp.Name()).To(Equal("test-app"))
		})

		It("should return an error for a non-existent application", func() {
			_, err := mockRepo.GetByName(context.Background(), "inexistante")
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, application.ErrApplicationNotFound)).To(BeTrue())
		})

		It("should list all applications", func() {
			app1, _ := application.NewApplication("app1")
			app2, _ := application.NewApplication("app2")

			err := mockRepo.Save(context.Background(), app1)
			Expect(err).NotTo(HaveOccurred())
			err = mockRepo.Save(context.Background(), app2)
			Expect(err).NotTo(HaveOccurred())

			apps, err := mockRepo.GetAll(context.Background())
			Expect(err).NotTo(HaveOccurred())
			Expect(apps).To(HaveLen(2))
		})

		It("should check the existence of applications", func() {
			app, _ := application.NewApplication("exists")
			err := mockRepo.Save(context.Background(), app)
			Expect(err).NotTo(HaveOccurred())

			exists, err := mockRepo.Exists(context.Background(), "exists")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())

			exists, err = mockRepo.Exists(context.Background(), "not-exists")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should filter by state", func() {
			app1, _ := application.NewApplication("running-app")
			app2, _ := application.NewApplication("stopped-app")

			// Follow valid state transitions: Created -> Deployed -> Running/Stopped
			Expect(app1.UpdateState(application.StateDeployed)).To(Succeed())
			Expect(app1.UpdateState(application.StateRunning)).To(Succeed())

			Expect(app2.UpdateState(application.StateDeployed)).To(Succeed())
			Expect(app2.UpdateState(application.StateStopped)).To(Succeed())

			err := mockRepo.Save(context.Background(), app1)
			Expect(err).NotTo(HaveOccurred())
			err = mockRepo.Save(context.Background(), app2)
			Expect(err).NotTo(HaveOccurred())

			runningApps, err := mockRepo.GetByState(context.Background(), application.StateRunning)
			Expect(err).NotTo(HaveOccurred())
			Expect(runningApps).To(HaveLen(1))
			Expect(runningApps[0].Name()).To(Equal("running-app"))
		})
	})

	Describe("Error simulation", func() {
		It("should simulate save errors", func() {
			testError := errors.New("test error")
			mockRepo.SetSaveError(testError)

			app, _ := application.NewApplication("test")
			err := mockRepo.Save(context.Background(), app)

			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(testError))
		})

		It("should simulate retrieval errors", func() {
			testError := errors.New("retrieval error")
			mockRepo.SetGetError(testError)

			_, err := mockRepo.GetAll(context.Background())
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(testError))
		})
	})
})

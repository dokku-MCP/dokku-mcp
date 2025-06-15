package application_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/alex-galey/dokku-mcp/src/domain/application"
)

var _ = Describe("Application Entity", func() {
	Describe("Creating a new application", func() {
		Context("with valid parameters", func() {
			It("should create an application with the initial state", func() {
				app, err := application.NewApplication("test-app")

				Expect(err).NotTo(HaveOccurred())
				Expect(app).NotTo(BeNil())
				Expect(app.Name()).To(Equal("test-app"))
				Expect(app.State()).To(Equal(application.StateCreated))
				Expect(app.Config()).NotTo(BeNil())
				Expect(app.CreatedAt()).To(BeTemporally("~", time.Now(), time.Second))
				Expect(app.UpdatedAt()).To(BeTemporally("~", time.Now(), time.Second))
				Expect(app.LastDeploy()).To(BeNil())
			})
		})

		Context("with invalid parameters", func() {
			It("should reject an empty name", func() {
				app, err := application.NewApplication("")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("application name cannot be empty"))
				Expect(app).To(BeNil())
			})

			It("should reject a name that is too long (> 63 characters)", func() {
				longName := "very-long-application-name-that-actually-exceeds-the-dns-label-limit-of-sixty-three-characters"
				app, err := application.NewApplication(longName)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("application name cannot exceed 63 characters"))
				Expect(app).To(BeNil())
			})
		})
	})

	Describe("Application state transitions", func() {
		var app *application.Application

		BeforeEach(func() {
			var err error
			app, err = application.NewApplication("test-app")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("valid state transitions", func() {
			It("should allow the transition from Created to Deployed", func() {
				err := app.UpdateState(application.StateDeployed)

				Expect(err).NotTo(HaveOccurred())
				Expect(app.State()).To(Equal(application.StateDeployed))
			})

			It("should allow the transition from Created to Error", func() {
				err := app.UpdateState(application.StateError)

				Expect(err).NotTo(HaveOccurred())
				Expect(app.State()).To(Equal(application.StateError))
			})

			It("should allow the transition from Deployed to Running", func() {
				Expect(app.UpdateState(application.StateDeployed)).To(Succeed())

				err := app.UpdateState(application.StateRunning)

				Expect(err).NotTo(HaveOccurred())
				Expect(app.State()).To(Equal(application.StateRunning))
			})
		})

		Context("invalid state transitions", func() {
			It("should reject the direct transition from Created to Running", func() {
				err := app.UpdateState(application.StateRunning)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid state transition"))
			})

			It("should reject an invalid state", func() {
				err := app.UpdateState(application.ApplicationState("invalid"))

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid application state"))
			})
		})

		It("should update the timestamp when the state changes", func() {
			initialTime := app.UpdatedAt()
			time.Sleep(time.Millisecond) // Ensure a time difference

			err := app.UpdateState(application.StateDeployed)

			Expect(err).NotTo(HaveOccurred())
			Expect(app.UpdatedAt()).To(BeTemporally(">", initialTime))
		})
	})

	Describe("Application deployment", func() {
		var app *application.Application

		BeforeEach(func() {
			var err error
			app, err = application.NewApplication("test-app")
			Expect(err).NotTo(HaveOccurred())
		})

		Context("with valid parameters", func() {
			It("should deploy the application successfully", func() {
				err := app.Deploy("main", "build-image:latest", "run-image:latest")

				Expect(err).NotTo(HaveOccurred())
				Expect(app.State()).To(Equal(application.StateDeployed))
				Expect(app.GitRef()).To(Equal("main"))
				Expect(app.BuildImage()).To(Equal("build-image:latest"))
				Expect(app.RunImage()).To(Equal("run-image:latest"))
				Expect(app.LastDeploy()).NotTo(BeNil())
				Expect(*app.LastDeploy()).To(BeTemporally("~", time.Now(), time.Second))
			})
		})

		Context("with invalid parameters", func() {
			It("should reject an empty git reference", func() {
				err := app.Deploy("", "build-image", "run-image")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("git reference cannot be empty"))
			})
		})
	})

	Describe("Utility methods", func() {
		var app *application.Application

		BeforeEach(func() {
			var err error
			app, err = application.NewApplication("test-app")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should correctly identify if the application is running", func() {
			Expect(app.IsRunning()).To(BeFalse())

			Expect(app.UpdateState(application.StateDeployed)).To(Succeed())
			Expect(app.IsRunning()).To(BeFalse())

			Expect(app.UpdateState(application.StateRunning)).To(Succeed())
			Expect(app.IsRunning()).To(BeTrue())
		})

		It("should correctly identify if the application is deployed", func() {
			Expect(app.IsDeployed()).To(BeFalse())

			Expect(app.UpdateState(application.StateDeployed)).To(Succeed())
			Expect(app.IsDeployed()).To(BeTrue())

			Expect(app.UpdateState(application.StateRunning)).To(Succeed())
			Expect(app.IsDeployed()).To(BeTrue())
		})
	})
})

var _ = Describe("ApplicationState", func() {
	Describe("State validation", func() {
		It("should validate valid states", func() {
			validStates := []application.ApplicationState{
				application.StateCreated,
				application.StateDeployed,
				application.StateRunning,
				application.StateStopped,
				application.StateError,
			}

			for _, state := range validStates {
				Expect(state.IsValid()).To(BeTrue(), "State %s should be valid", state)
			}
		})

		It("should reject invalid states", func() {
			invalidState := application.ApplicationState("invalid")
			Expect(invalidState.IsValid()).To(BeFalse())
		})
	})

	Describe("String conversion", func() {
		It("should convert correctly to string", func() {
			state := application.StateRunning
			Expect(state.String()).To(Equal("running"))
		})
	})
})

var _ = Describe("Process", func() {
	Describe("Creating a new process", func() {
		Context("with valid parameters", func() {
			It("should create a process successfully", func() {
				process, err := application.NewProcess(application.ProcessTypeWeb, "node server.js", 2)

				Expect(err).NotTo(HaveOccurred())
				Expect(process).NotTo(BeNil())
				Expect(process.Type).To(Equal(application.ProcessTypeWeb))
				Expect(process.Command).To(Equal("node server.js"))
				Expect(process.Scale).To(Equal(2))
			})
		})

		Context("with invalid parameters", func() {
			It("should reject an empty command", func() {
				process, err := application.NewProcess(application.ProcessTypeWeb, "", 1)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("process command cannot be empty"))
				Expect(process).To(BeNil())
			})

			It("should reject a negative scale", func() {
				process, err := application.NewProcess(application.ProcessTypeWeb, "node server.js", -1)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("process scale cannot be negative"))
				Expect(process).To(BeNil())
			})
		})
	})
})

var _ = Describe("ApplicationConfig", func() {
	var config *application.ApplicationConfig

	BeforeEach(func() {
		config = application.NewApplicationConfig()
	})

	Describe("Domain management", func() {
		Context("adding valid domains", func() {
			It("should add a domain successfully", func() {
				err := config.AddDomain("example.com")

				Expect(err).NotTo(HaveOccurred())
				Expect(config.Domains).To(ContainElement("example.com"))
			})

			It("should allow adding multiple domains", func() {
				Expect(config.AddDomain("example.com")).To(Succeed())
				Expect(config.AddDomain("test.example.com")).To(Succeed())

				Expect(config.Domains).To(HaveLen(2))
				Expect(config.Domains).To(ContainElements("example.com", "test.example.com"))
			})
		})

		Context("adding invalid domains", func() {
			It("should reject an empty domain", func() {
				err := config.AddDomain("")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("domain cannot be empty"))
			})

			It("should reject a domain that already exists", func() {
				Expect(config.AddDomain("example.com")).To(Succeed())

				err := config.AddDomain("example.com")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("domain example.com already exists"))
			})
		})
	})

	Describe("Environment variable management", func() {
		Context("adding valid environment variables", func() {
			It("should add an environment variable successfully", func() {
				err := config.SetEnvironmentVar("NODE_ENV", "production")

				Expect(err).NotTo(HaveOccurred())
				Expect(config.EnvironmentVars).To(HaveKeyWithValue("NODE_ENV", "production"))
			})

			It("should allow overriding an existing environment variable", func() {
				Expect(config.SetEnvironmentVar("NODE_ENV", "development")).To(Succeed())
				Expect(config.SetEnvironmentVar("NODE_ENV", "production")).To(Succeed())

				Expect(config.EnvironmentVars).To(HaveKeyWithValue("NODE_ENV", "production"))
			})
		})

		Context("adding invalid environment variables", func() {
			It("should reject an empty key", func() {
				err := config.SetEnvironmentVar("", "value")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("environment variable key cannot be empty"))
			})
		})
	})

	Describe("Process management", func() {
		Context("adding valid processes", func() {
			It("should add a process successfully", func() {
				process, err := application.NewProcess(application.ProcessTypeWeb, "node server.js", 1)
				Expect(err).NotTo(HaveOccurred())

				err = config.AddProcess(process)

				Expect(err).NotTo(HaveOccurred())
				Expect(config.Processes).To(HaveKeyWithValue(application.ProcessTypeWeb, process))
			})

			It("should allow overriding an existing process", func() {
				process1, err := application.NewProcess(application.ProcessTypeWeb, "node server.js", 1)
				Expect(err).NotTo(HaveOccurred())
				process2, err := application.NewProcess(application.ProcessTypeWeb, "npm start", 2)
				Expect(err).NotTo(HaveOccurred())

				Expect(config.AddProcess(process1)).To(Succeed())
				Expect(config.AddProcess(process2)).To(Succeed())

				Expect(config.Processes).To(HaveKeyWithValue(application.ProcessTypeWeb, process2))
			})
		})

		Context("adding invalid processes", func() {
			It("should reject a nil process", func() {
				err := config.AddProcess(nil)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("process cannot be nil"))
			})
		})
	})
})

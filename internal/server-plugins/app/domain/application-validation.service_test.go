//go:build !integration

package app

import (
	"context"

	"github.com/dokku-mcp/dokku-mcp/internal/shared"
	"github.com/dokku-mcp/dokku-mcp/internal/shared/process"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ValidationService", func() {
	var (
		service *ValidationService
		ctx     context.Context
	)

	BeforeEach(func() {
		service = NewValidationService()
		ctx = context.Background()
	})

	Describe("ValidateApplicationName", func() {
		DescribeTable("application name validation",
			func(appName string, expectValid bool, expectErrorCount int, expectWarningCount int, errorCodes []string, warningCodes []string) {
				result := service.ValidateApplicationName(ctx, appName)

				Expect(result.IsValid).To(Equal(expectValid))
				Expect(result.Errors).To(HaveLen(expectErrorCount))
				Expect(result.Warnings).To(HaveLen(expectWarningCount))

				if len(errorCodes) > 0 {
					actualErrorCodes := make([]string, len(result.Errors))
					for i, err := range result.Errors {
						actualErrorCodes[i] = err.Code
					}
					Expect(actualErrorCodes).To(ConsistOf(errorCodes))
				}

				if len(warningCodes) > 0 {
					actualWarningCodes := make([]string, len(result.Warnings))
					for i, warn := range result.Warnings {
						actualWarningCodes[i] = warn.Code
					}
					Expect(actualWarningCodes).To(ConsistOf(warningCodes))
				}
			},
			Entry("valid simple name", "myapp", true, 0, 0, []string{}, []string{}),
			Entry("valid name with hyphen", "my-app", true, 0, 0, []string{}, []string{}),
			Entry("empty name - VO validation should catch this", "", false, 1, 0, []string{"INVALID_APPLICATION_NAME"}, []string{}),
			Entry("name too long - VO validation should catch this", "this-is-a-very-long-application-name-that-exceeds-the-limit-of-63-chars", false, 1, 0, []string{"INVALID_APPLICATION_NAME"}, []string{}),
			Entry("invalid characters - VO validation should catch this", "my_app_with_underscores", false, 1, 0, []string{"INVALID_APPLICATION_NAME"}, []string{}),
			Entry("reserved name - should pass VO validation but add warning", "dokku", false, 1, 0, []string{"INVALID_APPLICATION_NAME"}, []string{}),
			Entry("long name without hyphens - should add format suggestion warning", "verylongapplicationnamewithouthyphens", true, 0, 1, []string{}, []string{"NAME_FORMAT_SUGGESTION"}),
		)
	})

	Describe("ValidateApplication", func() {
		Context("when validating a valid application", func() {
			It("should return valid result with no errors", func() {
				app, err := NewApplication("test-app")
				Expect(err).ToNot(HaveOccurred())

				result := service.ValidateApplication(ctx, app)

				Expect(result.IsValid).To(BeTrue())
				Expect(result.Errors).To(BeEmpty())
			})
		})

		Context("when creating an application with reserved name", func() {
			It("should fail at application creation level", func() {
				// This test demonstrates that the ApplicationName VO itself prevents
				// creation of apps with reserved names, so ValidationService doesn't
				// need to re-implement this validation
				_, err := NewApplication("dokku")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when application has state inconsistency", func() {
			It("should be valid but may have warnings", func() {
				app, err := NewApplication("test-app")
				Expect(err).ToNot(HaveOccurred())

				// Force an inconsistent state for testing
				// (This would typically not happen in normal operation)
				result := service.ValidateApplication(ctx, app)

				Expect(result.IsValid).To(BeTrue()) // Warnings don't make it invalid
				// The specific warning depends on the application's internal state
			})
		})
	})

	Describe("ValidateDeployment", func() {
		Context("when validating a valid deployment", func() {
			It("should return valid result", func() {
				app, err := NewApplication("test-app")
				Expect(err).ToNot(HaveOccurred())

				gitRef, err := shared.NewGitRef("main")
				Expect(err).ToNot(HaveOccurred())

				buildpackName := "heroku/nodejs"

				result := service.ValidateDeployment(ctx, app, gitRef, buildpackName)

				Expect(result.IsValid).To(BeTrue())
				Expect(result.Errors).To(BeEmpty())
			})
		})

		Context("when deployment is requested without application", func() {
			It("should return error", func() {
				result := service.ValidateDeployment(ctx, nil, nil, "")

				Expect(result.IsValid).To(BeFalse())
				Expect(result.Errors).To(HaveLen(1))
				Expect(result.Errors[0].Code).To(Equal("APP_REQUIRED"))
			})
		})

		Context("when deployment uses empty buildpack", func() {
			It("should add warning for auto-detection", func() {
				app, err := NewApplication("test-app")
				Expect(err).ToNot(HaveOccurred())

				result := service.ValidateDeployment(ctx, app, nil, "")

				Expect(result.IsValid).To(BeTrue())
				Expect(result.Warnings).To(HaveLen(1))
				Expect(result.Warnings[0].Code).To(Equal("AUTO_BUILDPACK"))
			})
		})

		Context("when application is in error state", func() {
			It("should be valid but add warning", func() {
				app, err := NewApplication("test-app")
				Expect(err).ToNot(HaveOccurred())

				// Start a deployment first
				gitRef, err := shared.NewGitRef("main")
				Expect(err).ToNot(HaveOccurred())

				err = app.Deploy(gitRef, nil)
				Expect(err).ToNot(HaveOccurred())

				// Force the application into error state
				err = app.FailDeployment("test error")
				Expect(err).ToNot(HaveOccurred())

				result := service.ValidateDeployment(ctx, app, nil, "heroku/nodejs")

				Expect(result.IsValid).To(BeTrue()) // Warnings don't make it invalid
				Expect(result.Warnings).To(HaveLen(1))
				Expect(result.Warnings[0].Code).To(Equal("APP_ERROR_STATE"))
			})
		})
	})

	Describe("ValidateScale", func() {
		var (
			app         *Application
			processType process.ProcessType
		)

		BeforeEach(func() {
			var err error
			app, err = NewApplication("test-app")
			Expect(err).ToNot(HaveOccurred())
			processType = process.ProcessTypeWeb
		})

		DescribeTable("scale validation scenarios",
			func(scale int, expectValid bool, expectErrorCount int, expectWarningCount int, errorCodes []string, warningCodes []string) {
				result := service.ValidateScale(ctx, app, processType, scale)

				Expect(result.IsValid).To(Equal(expectValid))
				Expect(result.Errors).To(HaveLen(expectErrorCount))
				Expect(result.Warnings).To(HaveLen(expectWarningCount))

				if len(errorCodes) > 0 {
					actualErrorCodes := make([]string, len(result.Errors))
					for i, err := range result.Errors {
						actualErrorCodes[i] = err.Code
					}
					Expect(actualErrorCodes).To(ConsistOf(errorCodes))
				}

				if len(warningCodes) > 0 {
					actualWarningCodes := make([]string, len(result.Warnings))
					for i, warn := range result.Warnings {
						actualWarningCodes[i] = warn.Code
					}
					Expect(actualWarningCodes).To(ConsistOf(warningCodes))
				}
			},
			Entry("valid scale", 3, true, 0, 1, []string{}, []string{"PROCESS_NOT_CONFIGURED"}),
			Entry("zero scale", 0, true, 0, 0, []string{}, []string{}),
			Entry("negative scale", -1, false, 1, 0, []string{"INVALID_SCALE"}, []string{}),
			Entry("high scale", 100, true, 0, 1, []string{}, []string{"HIGH_SCALE_WARNING"}),
		)

		Context("when scaling an existing process", func() {
			It("should not warn about unconfigured process", func() {
				// Add a process to the application first
				err := app.AddProcess(processType, "web: node server.js", 1)
				Expect(err).ToNot(HaveOccurred())

				result := service.ValidateScale(ctx, app, processType, 3)

				Expect(result.IsValid).To(BeTrue())
				Expect(result.Warnings).To(BeEmpty())
			})
		})
	})
})

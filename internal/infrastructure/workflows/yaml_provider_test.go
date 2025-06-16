package workflows_test

import (
	"os"
	"path/filepath"

	"github.com/alex-galey/dokku-mcp/internal/domain/workflows"
	infraWorkflows "github.com/alex-galey/dokku-mcp/internal/infrastructure/workflows"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("YAMLFileProvider", func() {
	var (
		provider    workflows.WorkflowProvider
		tempDir     string
		workflowDir string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "workflow-test-*")
		Expect(err).NotTo(HaveOccurred())

		workflowDir = filepath.Join(tempDir, "workflows")
		err = os.MkdirAll(workflowDir, 0755)
		Expect(err).NotTo(HaveOccurred())

		provider = infraWorkflows.NewYAMLFileProvider(workflowDir)
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Describe("GetWorkflows", func() {
		Context("when directory is empty", func() {
			It("should return empty slice", func() {
				workflows, err := provider.GetWorkflows()
				Expect(err).NotTo(HaveOccurred())
				Expect(workflows).To(BeEmpty())
			})
		})

		Context("when directory doesn't exist", func() {
			BeforeEach(func() {
				provider = infraWorkflows.NewYAMLFileProvider("/nonexistent/path")
			})

			It("should return empty slice without error", func() {
				workflows, err := provider.GetWorkflows()
				Expect(err).NotTo(HaveOccurred())
				Expect(workflows).To(BeEmpty())
			})
		})

		Context("when valid workflow files exist", func() {
			BeforeEach(func() {
				// Create a valid workflow file
				workflowContent := `name: test_workflow
description: A test workflow
owner_plugin: core
arguments:
  - name: app_name
    type: string
    description: Application name
    required: true

steps:
  - name: create_app
    type: tool_call
    target: create_application
    arguments:
      name: "${app_name}"
    output_alias: create_result
`
				workflowFile := filepath.Join(workflowDir, "test_workflow.yaml")
				err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should load and parse workflows correctly", func() {
				workflows, err := provider.GetWorkflows()
				Expect(err).NotTo(HaveOccurred())
				Expect(workflows).To(HaveLen(1))

				workflow := workflows[0]
				Expect(workflow.Name).To(Equal("test_workflow"))
				Expect(workflow.Description).To(Equal("A test workflow"))
				Expect(workflow.OwnerPlugin).To(Equal("core"))
				Expect(workflow.Arguments).To(HaveLen(1))
				Expect(workflow.Steps).To(HaveLen(1))

				// Check argument details
				arg := workflow.Arguments[0]
				Expect(arg.Name).To(Equal("app_name"))
				Expect(arg.Type).To(Equal("string"))
				Expect(arg.Required).To(BeTrue())

				// Check step details
				step := workflow.Steps[0]
				Expect(step.Name).To(Equal("create_app"))
				Expect(step.Type).To(Equal("tool_call"))
				Expect(step.Target).To(Equal("create_application"))
				Expect(step.OutputAlias).To(Equal("create_result"))
			})
		})

		Context("when invalid workflow files exist", func() {
			BeforeEach(func() {
				// Create an invalid workflow file (missing required fields)
				invalidContent := `name: invalid_workflow
# Missing description and steps
`
				workflowFile := filepath.Join(workflowDir, "invalid.yaml")
				err := os.WriteFile(workflowFile, []byte(invalidContent), 0644)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should return an error", func() {
				_, err := provider.GetWorkflows()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("workflow description is required"))
			})
		})

		Context("when non-YAML files exist", func() {
			BeforeEach(func() {
				// Create a non-YAML file
				textFile := filepath.Join(workflowDir, "readme.txt")
				err := os.WriteFile(textFile, []byte("This is not a workflow"), 0644)
				Expect(err).NotTo(HaveOccurred())

				// Create a valid workflow file
				workflowContent := `name: test_workflow
description: A test workflow
owner_plugin: core
arguments: []
steps:
  - name: test_step
    type: tool_call
    target: test_tool
    arguments: {}
`
				workflowFile := filepath.Join(workflowDir, "test.yaml")
				err = os.WriteFile(workflowFile, []byte(workflowContent), 0644)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should only load YAML files", func() {
				workflows, err := provider.GetWorkflows()
				Expect(err).NotTo(HaveOccurred())
				Expect(workflows).To(HaveLen(1))
				Expect(workflows[0].Name).To(Equal("test_workflow"))
			})
		})
	})

	Describe("GetWorkflow", func() {
		BeforeEach(func() {
			// Create a test workflow file
			workflowContent := `name: specific_workflow
description: A specific test workflow
owner_plugin: core
arguments: []
steps:
  - name: test_step
    type: tool_call
    target: test_tool
    arguments: {}
`
			workflowFile := filepath.Join(workflowDir, "specific.yaml")
			err := os.WriteFile(workflowFile, []byte(workflowContent), 0644)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when workflow exists", func() {
			It("should return the specific workflow", func() {
				workflow, err := provider.GetWorkflow("specific_workflow")
				Expect(err).NotTo(HaveOccurred())
				Expect(workflow).NotTo(BeNil())
				Expect(workflow.Name).To(Equal("specific_workflow"))
			})
		})

		Context("when workflow doesn't exist", func() {
			It("should return an error", func() {
				_, err := provider.GetWorkflow("nonexistent_workflow")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("workflow 'nonexistent_workflow' not found"))
			})
		})
	})
})

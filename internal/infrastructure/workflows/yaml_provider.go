package workflows

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/alex-galey/dokku-mcp/internal/domain/workflows"
	"github.com/goccy/go-yaml"
)

// YAMLFileProvider implements WorkflowProvider by reading workflow definitions
// from YAML files in a specified directory.
type YAMLFileProvider struct {
	workflowsDir string
}

// NewYAMLFileProvider creates a new YAMLFileProvider instance.
func NewYAMLFileProvider(workflowsDir string) workflows.WorkflowProvider {
	return &YAMLFileProvider{
		workflowsDir: workflowsDir,
	}
}

// GetWorkflows returns all workflow definitions found in YAML files.
func (p *YAMLFileProvider) GetWorkflows() ([]workflows.Workflow, error) {
	var workflowList []workflows.Workflow

	// Check if workflows directory exists
	if _, err := os.Stat(p.workflowsDir); os.IsNotExist(err) {
		// Return empty slice if directory doesn't exist
		return workflowList, nil
	}

	// Walk through all YAML files in the directory
	err := filepath.WalkDir(p.workflowsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-YAML files
		if d.IsDir() || !isYAMLFile(path) {
			return nil
		}

		// Read and parse the YAML file
		workflow, err := p.parseWorkflowFile(path)
		if err != nil {
			return fmt.Errorf("failed to parse workflow file %s: %w", path, err)
		}

		workflowList = append(workflowList, *workflow)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read workflows directory: %w", err)
	}

	return workflowList, nil
}

// GetWorkflow returns a specific workflow by name.
func (p *YAMLFileProvider) GetWorkflow(name string) (*workflows.Workflow, error) {
	workflowList, err := p.GetWorkflows()
	if err != nil {
		return nil, err
	}

	for _, workflow := range workflowList {
		if workflow.Name == name {
			return &workflow, nil
		}
	}

	return nil, fmt.Errorf("workflow '%s' not found", name)
}

// parseWorkflowFile reads and parses a single YAML workflow file.
func (p *YAMLFileProvider) parseWorkflowFile(filePath string) (*workflows.Workflow, error) {
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse YAML
	var workflow workflows.Workflow
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate required fields
	if workflow.Name == "" {
		return nil, fmt.Errorf("workflow name is required")
	}

	if workflow.Description == "" {
		return nil, fmt.Errorf("workflow description is required")
	}

	if len(workflow.Steps) == 0 {
		return nil, fmt.Errorf("workflow must have at least one step")
	}

	// Validate each step
	for i, step := range workflow.Steps {
		if step.Name == "" {
			return nil, fmt.Errorf("step %d: name is required", i)
		}

		if step.Type == "" {
			return nil, fmt.Errorf("step %d (%s): type is required", i, step.Name)
		}

		if step.Target == "" {
			return nil, fmt.Errorf("step %d (%s): target is required", i, step.Name)
		}

		// Validate step type
		if !isValidStepType(step.Type) {
			return nil, fmt.Errorf("step %d (%s): invalid type '%s'", i, step.Name, step.Type)
		}
	}

	return &workflow, nil
}

// isYAMLFile checks if a file has a YAML extension.
func isYAMLFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".yaml" || ext == ".yml"
}

// isValidStepType checks if a step type is valid.
func isValidStepType(stepType string) bool {
	validTypes := []string{
		"tool_call",
		"read_resource",
		"prompt",
		"conditional",
	}

	for _, validType := range validTypes {
		if stepType == validType {
			return true
		}
	}

	return false
}

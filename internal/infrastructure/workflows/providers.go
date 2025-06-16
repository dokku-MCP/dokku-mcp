package workflows

import (
	domainWorkflows "github.com/alex-galey/dokku-mcp/internal/domain/workflows"
)

// ProvideYAMLWorkflowProvider provides a YAML-based workflow provider.
func ProvideYAMLWorkflowProvider() domainWorkflows.WorkflowProvider {
	return NewYAMLFileProvider("workflows")
}

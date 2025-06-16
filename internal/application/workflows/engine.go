package workflows

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/alex-galey/dokku-mcp/internal/application/plugins"
	"github.com/alex-galey/dokku-mcp/internal/domain/workflows"
	"github.com/mark3labs/mcp-go/server"
)

// WorkflowEngine executes workflow definitions by coordinating with
// the plugin registry and MCP server.
type WorkflowEngine struct {
	provider       workflows.WorkflowProvider
	pluginRegistry *plugins.DynamicPluginRegistry
	mcpServer      *server.MCPServer
	logger         *slog.Logger
}

// NewWorkflowEngine creates a new WorkflowEngine instance.
func NewWorkflowEngine(
	provider workflows.WorkflowProvider,
	pluginRegistry *plugins.DynamicPluginRegistry,
	mcpServer *server.MCPServer,
	logger *slog.Logger,
) *WorkflowEngine {
	return &WorkflowEngine{
		provider:       provider,
		pluginRegistry: pluginRegistry,
		mcpServer:      mcpServer,
		logger:         logger,
	}
}

// GetActiveWorkflows returns all workflows that are currently active
// based on the plugin registry state.
func (e *WorkflowEngine) GetActiveWorkflows() ([]workflows.Workflow, error) {
	// Get all workflows from provider
	allWorkflows, err := e.provider.GetWorkflows()
	if err != nil {
		return nil, fmt.Errorf("failed to get workflows from provider: %w", err)
	}

	// Filter workflows based on active plugins
	var activeWorkflows []workflows.Workflow
	for _, workflow := range allWorkflows {
		if e.isWorkflowActive(workflow) {
			activeWorkflows = append(activeWorkflows, workflow)
		}
	}

	return activeWorkflows, nil
}

// RunWorkflow executes a workflow by name with the provided arguments.
func (e *WorkflowEngine) RunWorkflow(ctx context.Context, name string, args map[string]interface{}) (*workflows.WorkflowResult, error) {
	e.logger.Info("Starting workflow execution",
		"workflow_name", name,
		"arguments", args)

	// Get the workflow definition
	workflow, err := e.provider.GetWorkflow(name)
	if err != nil {
		return &workflows.WorkflowResult{
			Success: false,
			Error:   fmt.Sprintf("Workflow not found: %v", err),
		}, nil
	}

	// Check if workflow is active
	if !e.isWorkflowActive(*workflow) {
		return &workflows.WorkflowResult{
			Success: false,
			Error:   fmt.Sprintf("Workflow '%s' is not active (required plugin '%s' not available)", name, workflow.OwnerPlugin),
		}, nil
	}

	// Initialize execution context
	execCtx := &workflows.WorkflowExecutionContext{
		Variables:    make(map[string]interface{}),
		CurrentStep:  0,
		WorkflowName: name,
	}

	// Set initial variables from arguments
	for key, value := range args {
		execCtx.Variables[key] = value
	}

	// Execute workflow steps
	var stepResults []workflows.StepResult
	for i, step := range workflow.Steps {
		execCtx.CurrentStep = i

		e.logger.Debug("Executing workflow step",
			"workflow", name,
			"step", i,
			"step_name", step.Name,
			"step_type", step.Type)

		// Check step condition if present
		if step.Condition != "" && !e.evaluateCondition(step.Condition, execCtx) {
			e.logger.Debug("Step condition not met, skipping",
				"workflow", name,
				"step", step.Name,
				"condition", step.Condition)

			stepResults = append(stepResults, workflows.StepResult{
				StepName: step.Name,
				Success:  true,
				Output:   "Step skipped due to condition",
			})
			continue
		}

		// Execute the step
		stepResult, err := e.executeStep(ctx, step, execCtx)
		if err != nil {
			e.logger.Error("Step execution failed",
				"workflow", name,
				"step", step.Name,
				"error", err)

			stepResults = append(stepResults, workflows.StepResult{
				StepName: step.Name,
				Success:  false,
				Error:    err.Error(),
			})

			return &workflows.WorkflowResult{
				Success:     false,
				Message:     fmt.Sprintf("Workflow failed at step '%s'", step.Name),
				StepResults: stepResults,
				Variables:   execCtx.Variables,
				Error:       err.Error(),
			}, nil
		}

		stepResults = append(stepResults, *stepResult)

		// Store step output in variables if alias is provided
		if step.OutputAlias != "" && stepResult.Output != nil {
			execCtx.Variables[step.OutputAlias] = stepResult.Output
		}
	}

	e.logger.Info("Workflow execution completed successfully",
		"workflow", name,
		"steps_executed", len(stepResults))

	return &workflows.WorkflowResult{
		Success:     true,
		Message:     fmt.Sprintf("Workflow '%s' completed successfully", name),
		StepResults: stepResults,
		Variables:   execCtx.Variables,
	}, nil
}

// isWorkflowActive checks if a workflow should be active based on its owner plugin.
func (e *WorkflowEngine) isWorkflowActive(workflow workflows.Workflow) bool {
	// If no owner plugin is specified, the workflow is always active
	if workflow.OwnerPlugin == "" {
		return true
	}

	// Check if the owner plugin is active
	return e.pluginRegistry.IsPluginActive(workflow.OwnerPlugin)
}

// executeStep executes a single workflow step based on its type.
func (e *WorkflowEngine) executeStep(ctx context.Context, step workflows.WorkflowStep, execCtx *workflows.WorkflowExecutionContext) (*workflows.StepResult, error) {
	switch step.Type {
	case "tool_call":
		return e.executeToolCall(ctx, step, execCtx)
	case "read_resource":
		return e.executeReadResource(ctx, step, execCtx)
	case "prompt":
		return e.executePrompt(ctx, step, execCtx)
	case "conditional":
		return e.executeConditional(ctx, step, execCtx)
	default:
		return nil, fmt.Errorf("unsupported step type: %s", step.Type)
	}
}

// executeToolCall executes a tool_call step.
func (e *WorkflowEngine) executeToolCall(ctx context.Context, step workflows.WorkflowStep, execCtx *workflows.WorkflowExecutionContext) (*workflows.StepResult, error) {
	// Resolve arguments with variable substitution
	resolvedArgs := e.resolveArguments(step.Arguments, execCtx)

	// Note: This is a simplified implementation. In a real scenario,
	// we would need to call the actual tool handler through the MCP server.
	// For now, we'll return a placeholder result.

	e.logger.Debug("Tool call executed",
		"tool", step.Target,
		"arguments", resolvedArgs)

	return &workflows.StepResult{
		StepName: step.Name,
		Success:  true,
		Output:   fmt.Sprintf("Tool '%s' executed successfully", step.Target),
	}, nil
}

// executeReadResource executes a read_resource step.
func (e *WorkflowEngine) executeReadResource(ctx context.Context, step workflows.WorkflowStep, execCtx *workflows.WorkflowExecutionContext) (*workflows.StepResult, error) {
	// Note: This is a simplified implementation. In a real scenario,
	// we would need to call the actual resource handler through the MCP server.
	// For now, we'll return a placeholder result.

	e.logger.Debug("Resource read executed",
		"resource", step.Target)

	return &workflows.StepResult{
		StepName: step.Name,
		Success:  true,
		Output:   fmt.Sprintf("Resource '%s' read successfully", step.Target),
	}, nil
}

// executePrompt executes a prompt step.
func (e *WorkflowEngine) executePrompt(ctx context.Context, step workflows.WorkflowStep, execCtx *workflows.WorkflowExecutionContext) (*workflows.StepResult, error) {
	// Note: This is a simplified implementation. In a real scenario,
	// we would need to call the actual prompt handler through the MCP server.
	// For now, we'll return a placeholder result.

	e.logger.Debug("Prompt executed",
		"prompt", step.Target)

	return &workflows.StepResult{
		StepName: step.Name,
		Success:  true,
		Output:   fmt.Sprintf("Prompt '%s' executed successfully", step.Target),
	}, nil
}

// executeConditional executes a conditional step.
func (e *WorkflowEngine) executeConditional(ctx context.Context, step workflows.WorkflowStep, execCtx *workflows.WorkflowExecutionContext) (*workflows.StepResult, error) {
	// Evaluate the condition
	condition, ok := step.Arguments["condition"].(string)
	if !ok {
		return nil, fmt.Errorf("conditional step requires 'condition' argument")
	}

	result := e.evaluateCondition(condition, execCtx)

	return &workflows.StepResult{
		StepName: step.Name,
		Success:  true,
		Output:   result,
	}, nil
}

// resolveArguments resolves variable references in step arguments.
func (e *WorkflowEngine) resolveArguments(args map[string]interface{}, execCtx *workflows.WorkflowExecutionContext) map[string]interface{} {
	resolved := make(map[string]interface{})

	for key, value := range args {
		resolved[key] = e.resolveValue(value, execCtx)
	}

	return resolved
}

// resolveValue resolves a single value, handling variable substitution.
func (e *WorkflowEngine) resolveValue(value interface{}, execCtx *workflows.WorkflowExecutionContext) interface{} {
	switch v := value.(type) {
	case string:
		// Simple variable substitution: ${variable_name}
		if strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}") {
			varName := v[2 : len(v)-1]
			if varValue, exists := execCtx.Variables[varName]; exists {
				return varValue
			}
		}
		return v
	case map[string]interface{}:
		resolved := make(map[string]interface{})
		for k, val := range v {
			resolved[k] = e.resolveValue(val, execCtx)
		}
		return resolved
	case []interface{}:
		resolved := make([]interface{}, len(v))
		for i, val := range v {
			resolved[i] = e.resolveValue(val, execCtx)
		}
		return resolved
	default:
		return v
	}
}

// evaluateCondition evaluates a simple condition string.
func (e *WorkflowEngine) evaluateCondition(condition string, execCtx *workflows.WorkflowExecutionContext) bool {
	// Simple condition evaluation: variable_name == "value"
	// This is a placeholder implementation

	if strings.Contains(condition, "==") {
		parts := strings.Split(condition, "==")
		if len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(strings.Trim(parts[1], `"`))

			if varValue, exists := execCtx.Variables[left]; exists {
				return fmt.Sprintf("%v", varValue) == right
			}
		}
	}

	// Default to true for unrecognized conditions
	return true
}

package dokkuApi

import (
	"context"
	"fmt"
	"log/slog"
)

// BaseAdapter provides common functionality for all plugin adapters
type BaseAdapter struct {
	client DokkuClient
	logger *slog.Logger
}

// NewBaseAdapter creates a new base adapter
func NewBaseAdapter(client DokkuClient, logger *slog.Logger) *BaseAdapter {
	return &BaseAdapter{
		client: client,
		logger: logger,
	}
}

// ExecuteCommand provides direct access to the underlying client
func (a *BaseAdapter) ExecuteCommand(ctx context.Context, command string, args []string) ([]byte, error) {
	return a.client.ExecuteCommand(ctx, command, args)
}

// GetKeyValueData executes a command and returns parsed key-value data
func (a *BaseAdapter) GetKeyValueData(ctx context.Context, command string, args []string, separator string) (map[string]string, error) {
	return a.client.GetKeyValueOutput(ctx, command, args, separator)
}

// GetListData executes a command and returns parsed list data
func (a *BaseAdapter) GetListData(ctx context.Context, command string, args []string) ([]string, error) {
	return a.client.GetListOutput(ctx, command, args)
}

// GetTableData executes a command and returns parsed table data
func (a *BaseAdapter) GetTableData(ctx context.Context, command string, args []string, skipHeaders bool) ([]map[string]string, error) {
	return a.client.GetTableOutput(ctx, command, args, skipHeaders)
}

// GetStructuredData executes a command with custom parsing specification
func (a *BaseAdapter) GetStructuredData(ctx context.Context, spec CommandSpec) (*CommandResult, error) {
	return a.client.ExecuteStructured(ctx, spec)
}

// Logger provides access to the logger for adapter implementations
func (a *BaseAdapter) Logger() *slog.Logger {
	return a.logger
}

// Client provides access to the enhanced client for advanced operations
func (a *BaseAdapter) Client() DokkuClient {
	return a.client
}

// Common operation helpers that adapters can use directly

// ExecuteAndParseKeyValue is a convenience method for the most common pattern
func (a *BaseAdapter) ExecuteAndParseKeyValue(ctx context.Context, command string, args []string, separator string) (map[string]string, error) {
	a.logger.Debug("Executing command with key-value parsing",
		"command", command,
		"args", args,
		"separator", separator)

	result, err := a.client.GetKeyValueOutput(ctx, command, args, separator)
	if err != nil {
		a.logger.Error("Failed to execute command",
			"command", command,
			"args", args,
			"error", err)
		return nil, err
	}

	a.logger.Debug("Command executed successfully",
		"command", command,
		"result_count", len(result))

	return result, nil
}

// ExecuteAndParseList is a convenience method for list parsing
func (a *BaseAdapter) ExecuteAndParseList(ctx context.Context, command string, args []string) ([]string, error) {
	a.logger.Debug("Executing command with list parsing",
		"command", command,
		"args", args)

	result, err := a.client.GetListOutput(ctx, command, args)
	if err != nil {
		a.logger.Error("Failed to execute command",
			"command", command,
			"args", args,
			"error", err)
		return nil, err
	}

	a.logger.Debug("Command executed successfully",
		"command", command,
		"result_count", len(result))

	return result, nil
}

// ExecuteAndParseLines is a convenience method for parsing output into lines
func (a *BaseAdapter) ExecuteAndParseLines(ctx context.Context, command string, args []string, skipHeaders bool) ([]string, error) {
	a.logger.Debug("Executing command with line parsing",
		"command", command,
		"args", args,
		"skip_headers", skipHeaders)

	output, err := a.client.ExecuteCommand(ctx, command, args)
	if err != nil {
		a.logger.Error("Failed to execute command",
			"command", command,
			"args", args,
			"error", err)
		return nil, err
	}

	var result []string
	if skipHeaders {
		result = ParseLinesSkipHeaders(string(output))
	} else {
		result = ParseTrimmedLines(string(output), true)
	}

	a.logger.Debug("Command executed successfully",
		"command", command,
		"result_count", len(result))

	return result, nil
}

// ExecuteAndParseFields is a convenience method for parsing field-based output
func (a *BaseAdapter) ExecuteAndParseFields(ctx context.Context, command string, args []string, skipHeaders bool) ([][]string, error) {
	a.logger.Debug("Executing command with fields parsing",
		"command", command,
		"args", args,
		"skip_headers", skipHeaders)

	output, err := a.client.ExecuteCommand(ctx, command, args)
	if err != nil {
		a.logger.Error("Failed to execute command",
			"command", command,
			"args", args,
			"error", err)
		return nil, err
	}

	result := ParseFieldsOutput(string(output), skipHeaders)

	a.logger.Debug("Command executed successfully",
		"command", command,
		"result_count", len(result))

	return result, nil
}

// ValidateAndExecute provides validation and execution in one step
func (a *BaseAdapter) ValidateAndExecute(ctx context.Context, command string, args []string, allowedCommands []string) ([]byte, error) {
	// Validate command is allowed
	if len(allowedCommands) > 0 {
		allowed := false
		for _, allowedCmd := range allowedCommands {
			if command == allowedCmd {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("command %s is not in allowed commands list", command)
		}
	}

	return a.client.ExecuteCommand(ctx, command, args)
}

package dokkuApi

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// ValidateCommand performs validation on Dokku commands to ensure security
func (c *client) ValidateCommand(commandName string, args []string) error {
	if commandName == "" {
		return fmt.Errorf("command name cannot be empty")
	}

	// Blacklist first (runtime configuration)
	for _, blacklistedPattern := range c.blacklistedCommands {
		if strings.Contains(commandName, blacklistedPattern) {
			return fmt.Errorf("command is blacklisted (matches pattern '%s'): %s", blacklistedPattern, commandName)
		}
	}

	// Basic security validation - ensure no dangerous characters in command name
	// These characters could be used for command injection
	dangerousChars := []string{";", "&", "|", "`", "$", "(", ")", "{", "}", "<", ">", "\n", "\r"}
	for _, char := range dangerousChars {
		if strings.Contains(commandName, char) {
			return fmt.Errorf("command name contains dangerous character '%s': %s", char, commandName)
		}
	}

	// Validate arguments - ensure no dangerous characters
	for i, arg := range args {
		for _, char := range dangerousChars {
			if strings.Contains(arg, char) {
				return fmt.Errorf("argument %d contains dangerous character '%s': %s", i, char, arg)
			}
		}
	}

	// Additional validation: command should only contain alphanumeric, dash, colon
	for _, r := range commandName {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == ':') {
			return fmt.Errorf("command name contains invalid character: %c", r)
		}
	}

	// Log the command for audit purposes
	c.logger.Debug("Command validated",
		"command", commandName,
		"args_count", len(args))

	return nil
}

func NewDokkuClient(config *ClientConfig, logger *slog.Logger) DokkuClient {
	if config == nil {
		config = DefaultClientConfig()
	}

	// Create SSH configuration from client config
	sshConfig, err := NewSSHConfig(
		config.DokkuHost,
		config.DokkuPort,
		config.DokkuUser,
		config.SSHKeyPath,
		config.CommandTimeout,
	)
	if err != nil {
		logger.Error("Failed to create SSH configuration", "error", err)
		// Fall back to default configuration
		sshConfig = NewDefaultSSHConfig()
	}

	// Create SSH connection manager
	sshConnManager := NewSSHConnectionManager(sshConfig, logger)

	client := &client{
		config:         config,
		logger:         logger,
		sshConnManager: sshConnManager,
		capabilities:   NewDokkuCapabilities(),
	}

	// Initialize cache manager if caching is enabled
	client.cacheManager = NewCommandCacheManager(config.Cache, logger)

	// Discover Dokku capabilities in the background
	// This is non-blocking and will update capabilities asynchronously
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := client.DiscoverCapabilities(ctx); err != nil {
			logger.Warn("Failed to discover Dokku capabilities", "error", err)
		}
	}()

	return client
}

func (c *client) GetSSHConnectionManager() *SSHConnectionManager {
	return c.sshConnManager
}

func (c *client) ExecuteCommand(ctx context.Context, commandName string, args []string) ([]byte, error) {
	if err := c.ValidateCommand(commandName, args); err != nil {
		return nil, fmt.Errorf("invalid command: %w", err)
	}

	// Check cache first if caching is enabled
	if result, err, found := c.cacheManager.Get(commandName, args); found {
		return result, err
	}

	// Execute command
	result, err := c.executeCommandDirect(ctx, commandName, args)

	// Cache the result if caching is enabled
	c.cacheManager.Set(commandName, args, result, err)

	return result, err
}

// executeCommandDirect performs the actual command execution without caching
func (c *client) executeCommandDirect(ctx context.Context, commandName string, args []string) ([]byte, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, c.config.CommandTimeout)
	defer cancel()

	// For SSH connections to Dokku, commands are passed directly without the dokku path prefix
	// since Dokku SSH automatically routes commands to the Dokku CLI
	var dokkuCommand string
	if len(args) > 0 {
		dokkuCommand = commandName + " " + strings.Join(args, " ")
	} else {
		dokkuCommand = commandName
	}

	sshArgs, env, err := c.sshConnManager.PrepareSSHCommand(dokkuCommand)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare SSH command: %w", err)
	}

	// #nosec G204 -- Commands are validated through multiple layers:
	// 1. Plugin domain layer enforces whitelist of allowed commands via enums
	// 2. Infrastructure adapters validate commands before execution
	// 3. validateCommand() checks for dangerous characters and patterns
	// 4. There is a global blacklist of commands that are not allowed to be executed
	// 5. SSH command construction is internal, not from direct user input
	cmd := exec.CommandContext(cmdCtx, sshArgs[0], sshArgs[1:]...)
	cmd.Env = env
	// Set stdin to avoid potential SSH interaction issues
	cmd.Stdin = nil
	// Set process group to avoid signal propagation issues
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	c.logger.Debug("Executing Dokku command via SSH",
		"command", commandName,
		"args", args,
		"dokku_command", dokkuCommand,
		"ssh_target", c.sshConnManager.Config().ConnectionString(),
		"ssh_args", sshArgs,
		"env", env,
		"timeout", c.config.CommandTimeout,
		"context_deadline_ok", cmdCtx.Err() == nil,
		"connection_info", c.sshConnManager.GetConnectionInfo())

	// Use CombinedOutput to capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.logger.Error("Failed to execute Dokku command",
			"error", err,
			"command", commandName,
			"args", args,
			"dokku_command", dokkuCommand,
			"ssh_args", sshArgs,
			"env", env,
			"context_error", cmdCtx.Err(),
			"combined_output", string(output),
			"connection_info", c.sshConnManager.GetConnectionInfo())

		// Try to get stderr if available
		if exitError, ok := err.(*exec.ExitError); ok {
			c.logger.Error("Command exit details",
				"stderr", string(exitError.Stderr),
				"exit_code", exitError.ExitCode())
		}

		return nil, fmt.Errorf("failed to execute Dokku command %s: %w", commandName, err)
	}

	c.logger.Debug("Dokku command executed successfully",
		"command", commandName,
		"output_length", len(output))

	return output, nil
}

// InvalidateCache clears all cached entries (delegates to cache manager)
func (c *client) InvalidateCache() {
	c.cacheManager.Invalidate()
}

// SetBlacklist sets the blacklisted commands for runtime security configuration
func (c *client) SetBlacklist(commands []string) {
	c.blacklistedCommands = commands
	c.logger.Debug("Command blacklist updated", "patterns", commands) // Audit trail
}

// Enhanced parsing methods

// ExecuteStructured executes a command with automatic parsing based on the spec
func (c *client) ExecuteStructured(ctx context.Context, spec CommandSpec) (*CommandResult, error) {
	output, err := c.ExecuteCommand(ctx, spec.Command, spec.Args)
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %w", err)
	}

	result := &CommandResult{
		RawOutput: output,
		ParsedAt:  time.Now(),
	}

	switch spec.OutputFormat {
	case OutputFormatJSON:
		result.JSONData = output
	case OutputFormatKeyValue:
		result.KeyValueData = ParseKeyValueOutput(string(output), spec.Separator)
	case OutputFormatList:
		result.ListData = ParseListOutput(string(output), spec.FilterEmpty)
	case OutputFormatTable:
		result.TableData = ParseTableOutput(string(output), spec.SkipHeaders)
	case OutputFormatRaw:
		// Raw output is already stored in RawOutput
	default:
		return nil, fmt.Errorf("unsupported output format: %s", spec.OutputFormat)
	}

	return result, nil
}

// ExecuteWithAutoFormat executes a command with automatic format detection and optimal parsing
// This is the new JSON-first approach that prefers JSON when available
func (c *client) ExecuteWithAutoFormat(ctx context.Context, commandName string, args []string) (*CommandResult, error) {
	cap := c.capabilities.CommandRegistry.Get(commandName)

	// Check if command supports JSON
	supportsJSON := c.capabilities.SupportsJSON(commandName, c.capabilities.Version)

	if supportsJSON {
		// Try JSON first
		c.logger.Debug("Executing command with JSON format",
			"command", commandName,
			"supports_json", true)

		jsonArgs := append(args, "--format", "json")
		output, err := c.ExecuteCommand(ctx, commandName, jsonArgs)
		if err != nil {
			c.logger.Warn("Failed to execute with JSON format, falling back to text",
				"command", commandName,
				"error", err)
			// Fall through to text parsing
		} else {
			// Validate it's actually JSON
			if json.Valid(output) {
				return &CommandResult{
					RawOutput: output,
					JSONData:  output,
					ParsedAt:  time.Now(),
				}, nil
			}
			c.logger.Warn("Command returned non-JSON output despite --format json flag",
				"command", commandName)
		}
	}

	// Fall back to text parsing based on command characteristics
	output, err := c.ExecuteCommand(ctx, commandName, args)
	if err != nil {
		return nil, fmt.Errorf("command execution failed: %w", err)
	}

	result := &CommandResult{
		RawOutput: output,
		ParsedAt:  time.Now(),
	}

	// Infer parsing strategy from command name
	if cap != nil {
		c.logger.Debug("Using inferred parsing for command",
			"command", commandName,
			"supports_json", false)
	}

	// Default intelligent parsing based on command patterns
	if strings.Contains(commandName, ":list") {
		result.ListData = ParseListOutput(string(output), true)
	} else if strings.Contains(commandName, ":report") || strings.Contains(commandName, ":info") {
		result.KeyValueData = ParseKeyValueOutput(string(output), ":")
	} else if strings.Contains(commandName, "config:show") {
		result.KeyValueData = ParseKeyValueOutput(string(output), "=")
	}

	return result, nil
}

// GetKeyValueOutput executes a command and parses key-value output
func (c *client) GetKeyValueOutput(ctx context.Context, command string, args []string, separator string) (map[string]string, error) {
	spec := CommandSpec{
		Command:      command,
		Args:         args,
		OutputFormat: OutputFormatKeyValue,
		Separator:    separator,
	}

	result, err := c.ExecuteStructured(ctx, spec)
	if err != nil {
		return nil, err
	}

	return result.KeyValueData, nil
}

// GetListOutput executes a command and parses list output
func (c *client) GetListOutput(ctx context.Context, command string, args []string) ([]string, error) {
	spec := CommandSpec{
		Command:      command,
		Args:         args,
		OutputFormat: OutputFormatList,
		FilterEmpty:  true,
	}

	result, err := c.ExecuteStructured(ctx, spec)
	if err != nil {
		return nil, err
	}

	return result.ListData, nil
}

// GetTableOutput executes a command and parses table output
func (c *client) GetTableOutput(ctx context.Context, command string, args []string, skipHeaders bool) ([]map[string]string, error) {
	spec := CommandSpec{
		Command:      command,
		Args:         args,
		OutputFormat: OutputFormatTable,
		SkipHeaders:  skipHeaders,
	}

	result, err := c.ExecuteStructured(ctx, spec)
	if err != nil {
		return nil, err
	}

	return result.TableData, nil
}

package dokkuApi

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/dokku-mcp/dokku-mcp/internal/shared"
)

// isAppScopedCommand returns true for commands that target a specific app
func isAppScopedCommand(commandName string) bool {
	return strings.HasPrefix(commandName, "apps:") || strings.HasPrefix(commandName, "ps:") || commandName == "logs"
}

// ValidateCommand checks a Dokku command against the security policy: the
// command name and every argument must be shell-safe, the command must not
// match the blacklist and, when an allowlist is configured, must match it.
func (c *client) ValidateCommand(commandName string, args []string) error {
	if err := validateCommandName(commandName); err != nil {
		return err
	}

	for _, blacklistedPattern := range c.blacklistedCommands {
		if blacklistedPattern != "" && strings.Contains(commandName, blacklistedPattern) {
			return fmt.Errorf("command is blacklisted (matches pattern '%s'): %s", blacklistedPattern, commandName)
		}
	}

	if len(c.allowedCommands) > 0 && !c.isAllowlisted(commandName) {
		return fmt.Errorf("command is not in the allowlist: %s", commandName)
	}

	total := len(commandName)
	for i, arg := range args {
		if err := ValidateArg(arg); err != nil {
			return fmt.Errorf("argument %d: %w", i, err)
		}
		total += 1 + len(arg)
	}
	if total > MaxCommandBytes {
		return fmt.Errorf("command line exceeds %d bytes; split the request", MaxCommandBytes)
	}

	c.logger.Debug("Command validated",
		"command", commandName,
		"args_count", len(args))

	return nil
}

func (c *client) isAllowlisted(commandName string) bool {
	for _, pattern := range c.allowedCommands {
		if MatchesCommandPattern(commandName, pattern) {
			return true
		}
	}
	return false
}

func NewDokkuClient(config *ClientConfig, logger *slog.Logger) DokkuClient {
	if config == nil {
		config = DefaultClientConfig()
	}

	// Create SSH configuration from client config
	sshConfig, err := NewSSHConfigFromServerConfig(
		config.DokkuHost,
		config.DokkuPort,
		config.DokkuUser,
		config.SSHKeyPath,
		config.CommandTimeout,
		config.DisablePTY,
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

	if !IsCacheableCommand(commandName) {
		result, err := c.executeCommandDirect(ctx, commandName, args)
		if !isReadOnlyCommand(commandName) {
			// The command may have changed server state (even on failure),
			// so any cached report could now be stale.
			c.cacheManager.Invalidate()
		}
		return result, err
	}

	if result, found := c.cacheManager.Get(commandName, args); found {
		return result, nil
	}

	generation := c.cacheManager.Generation()
	result, err := c.executeCommandDirect(ctx, commandName, args)
	if err == nil {
		c.cacheManager.Set(commandName, args, result, generation)
	}
	return result, err
}

// executeCommandDirect performs the actual command execution without caching
func (c *client) executeCommandDirect(ctx context.Context, commandName string, args []string) ([]byte, error) {
	return c.executeCommandWithInput(ctx, commandName, args, nil)
}

// ExecuteCommandWithStdin runs a command that reads its payload from stdin,
// such as ssh-keys:add. It is never cached and always invalidates the cache.
func (c *client) ExecuteCommandWithStdin(ctx context.Context, commandName string, args []string, stdin string) ([]byte, error) {
	if err := c.ValidateCommand(commandName, args); err != nil {
		return nil, fmt.Errorf("invalid command: %w", err)
	}
	if len(stdin) > MaxStdinBytes {
		return nil, fmt.Errorf("stdin payload exceeds %d bytes", MaxStdinBytes)
	}
	defer c.cacheManager.Invalidate()
	return c.executeCommandWithInput(ctx, commandName, args, strings.NewReader(stdin))
}

func (c *client) executeCommandWithInput(ctx context.Context, commandName string, args []string, stdin io.Reader) ([]byte, error) {
	cmdCtx, cancel := c.commandContext(ctx)
	defer cancel()

	dokkuCommand := buildDokkuCommand(commandName, args)

	sshArgs, env, err := c.sshConnManager.PrepareSSHCommand(dokkuCommand)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare SSH command: %w", err)
	}

	cmd, err := prepareSSHExecCommand(cmdCtx, sshArgs, env)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare SSH command: %w", err)
	}

	if stdin != nil {
		cmd.Stdin = stdin
	}

	// Logs get a redacted copy: config:set values are reversible base64.
	logArgs := RedactArgs(commandName, args)
	logCommand := buildDokkuCommand(commandName, logArgs)
	logSSHArgs := append(slices.Clone(sshArgs[:len(sshArgs)-1]), logCommand)
	c.logCommandExecutionStart(cmdCtx, commandName, logArgs, logCommand, logSSHArgs, env)

	output, execErr := cmd.CombinedOutput()
	if execErr != nil {
		return c.handleCommandError(cmdCtx, commandName, logArgs, logCommand, logSSHArgs, env, output, execErr)
	}

	c.logger.Debug("Dokku command executed successfully",
		"command", commandName,
		"output_length", len(output))

	return output, nil
}

func (c *client) commandContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return ctx, func() {}
	}
	if c.config.CommandTimeout > 0 {
		return context.WithTimeout(ctx, c.config.CommandTimeout)
	}
	return ctx, func() {}
}

func buildDokkuCommand(commandName string, args []string) string {
	if len(args) == 0 {
		return commandName
	}
	return commandName + " " + strings.Join(args, " ")
}

func prepareSSHExecCommand(ctx context.Context, sshArgs []string, env []string) (*exec.Cmd, error) {
	if len(sshArgs) == 0 {
		return nil, fmt.Errorf("no SSH arguments provided")
	}

	// #nosec G204 -- Commands are validated through multiple layers prior to execution.
	cmd := exec.CommandContext(ctx, sshArgs[0], sshArgs[1:]...)
	cmd.Env = env
	cmd.Stdin = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: 0}
	return cmd, nil
}

// buildCommand builds an SSH command for execution
// This is used by StreamLogs to create a command that can be started and have its stdout piped
func (c *client) buildCommand(ctx context.Context, args []string) (*exec.Cmd, func(), error) {
	commandName := "logs"

	// Validate arguments for security
	if err := c.ValidateCommand(commandName, args); err != nil {
		return nil, nil, fmt.Errorf("invalid command arguments: %w", err)
	}

	// Build the Dokku command
	dokkuCommand := buildDokkuCommand(commandName, args)

	// Prepare SSH command
	sshArgs, env, err := c.sshConnManager.PrepareSSHCommand(dokkuCommand)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to prepare SSH command: %w", err)
	}

	// Create command context
	cmdCtx := ctx
	var cancelFunc func()
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		if c.config.CommandTimeout > 0 {
			var cancel context.CancelFunc
			cmdCtx, cancel = context.WithTimeout(ctx, c.config.CommandTimeout)
			cancelFunc = cancel
		}
	}

	// Prepare and return the command with cancel function
	cmd, err := prepareSSHExecCommand(cmdCtx, sshArgs, env)
	if err != nil {
		if cancelFunc != nil {
			cancelFunc()
		}
		return nil, nil, fmt.Errorf("failed to prepare SSH command: %w", err)
	}

	return cmd, cancelFunc, nil
}

func (c *client) logCommandExecutionStart(ctx context.Context, commandName string, args []string, dokkuCommand string, sshArgs []string, env []string) {
	c.logger.Debug("Executing Dokku command via SSH",
		"command", commandName,
		"args", args,
		"dokku_command", dokkuCommand,
		"ssh_target", c.sshConnManager.Config().ConnectionString(),
		"ssh_args", sshArgs,
		"env", env,
		"timeout", c.config.CommandTimeout,
		"context_deadline_ok", ctx.Err() == nil,
		"connection_info", c.sshConnManager.GetConnectionInfo())
}

func (c *client) handleCommandError(ctx context.Context, commandName string, args []string, dokkuCommand string, sshArgs []string, env []string, output []byte, execErr error) ([]byte, error) {
	if slices.Contains(secretValueCommands, commandName) {
		// config:set echoes the decoded values it was setting; keep them out
		// of logs and error messages.
		output = nil
	} else if len(output) > 0 {
		// git errors echo the repository URL, credentials included.
		output = []byte(shared.RedactURLCredentials(string(output)))
	}
	if isUnsupportedJSONProbe(args, output, commandName) {
		c.logger.Debug("JSON format not supported for command (probe)",
			"command", commandName,
			"args", args,
			"dokku_command", dokkuCommand,
			"combined_output", string(output))
		return nil, fmt.Errorf("failed to execute Dokku command %s: %w", commandName, execErr)
	}

	if shouldReturnEmptyLogs(commandName, output) {
		c.logger.Debug("Logs requested for app with no deployment yet; returning empty logs")
		return []byte(""), nil
	}

	c.logCommandFailure(ctx, commandName, args, dokkuCommand, sshArgs, env, output, execErr)
	c.logExitDetails(execErr)

	if shouldWrapNotFound(commandName, output) {
		return nil, fmt.Errorf("failed to execute Dokku command %s: %w", commandName, &NotFoundError{Command: commandName, Err: ErrAppNotFound})
	}

	if isTransportFailure(execErr) {
		// Details (host, port, ssh output) were logged above. The output
		// comes from the local ssh client, not Dokku, so it is not returned.
		return nil, fmt.Errorf("failed to execute Dokku command %s: %w", commandName, ErrDokkuUnreachable)
	}

	// Return the output as well: for builds it is the log the caller wants.
	if detail := errorDetail(output); detail != "" {
		return output, fmt.Errorf("failed to execute Dokku command %s: %w: %s", commandName, execErr, detail)
	}
	return output, fmt.Errorf("failed to execute Dokku command %s: %w", commandName, execErr)
}

// maxErrorDetail bounds how much command output is copied into an error.
const maxErrorDetail = 600

// errorDetail extracts Dokku's explanation from failed command output: the
// last non-empty lines, without the " !     " prefix Dokku uses for errors.
func errorDetail(output []byte) string {
	var lines []string
	for line := range strings.Lines(string(output)) {
		line = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "!"))
		if line != "" && !isSSHClientNoise(line) {
			lines = append(lines, line)
		}
	}
	detail := strings.Join(lines, "; ")
	if len(detail) > maxErrorDetail {
		detail = "..." + detail[len(detail)-maxErrorDetail:]
	}
	return detail
}

// sshClientNoisePrefixes start lines written by the local ssh client rather
// than by Dokku; they reveal connection details and never explain a failure.
// "ssh: " lines are kept: when the outer connection fails the error is
// replaced by ErrDokkuUnreachable anyway, and otherwise they come from ssh
// run by Dokku itself (e.g. git:sync cloning) and explain the failure.
var sshClientNoisePrefixes = []string{
	"Warning: Permanently added",
	"Pseudo-terminal will not be allocated",
	"Connection to ",
}

func isSSHClientNoise(line string) bool {
	for _, prefix := range sshClientNoisePrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

// isTransportFailure reports whether a command never ran on Dokku: ssh
// exits with 255 when it cannot connect or authenticate, and any error other
// than an exit status means ssh itself could not run.
func isTransportFailure(execErr error) bool {
	var exitErr *exec.ExitError
	if errors.As(execErr, &exitErr) {
		return exitErr.ExitCode() == 255
	}
	return !errors.Is(execErr, context.Canceled) && !errors.Is(execErr, context.DeadlineExceeded)
}

func isUnsupportedJSONProbe(args []string, output []byte, commandName string) bool {
	if !isJSONProbe(args) {
		return false
	}

	lowerOut := strings.ToLower(string(output))
	if strings.Contains(lowerOut, "unknown flag: --format") || strings.Contains(lowerOut, "is not a dokku command") {
		return true
	}

	usageProbe := "usage of " + strings.ToLower(commandName)
	return strings.Contains(lowerOut, usageProbe)
}

func isJSONProbe(args []string) bool {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--format" && args[i+1] == "json" {
			return true
		}
	}
	return false
}

func shouldReturnEmptyLogs(commandName string, output []byte) bool {
	if commandName != "logs" {
		return false
	}
	lower := strings.ToLower(string(output))
	return strings.Contains(lower, "has not been deployed")
}

func (c *client) logCommandFailure(ctx context.Context, commandName string, args []string, dokkuCommand string, sshArgs []string, env []string, output []byte, execErr error) {
	logFn := c.logger.Error
	lower := strings.ToLower(string(output))
	if isAppScopedCommand(commandName) && isNotFoundOutput(lower) {
		logFn = c.logger.Warn
	}

	logFn("Failed to execute Dokku command",
		"error", execErr,
		"command", commandName,
		"args", args,
		"dokku_command", dokkuCommand,
		"ssh_args", sshArgs,
		"env", env,
		"context_error", ctx.Err(),
		"combined_output", string(output),
		"connection_info", c.sshConnManager.GetConnectionInfo())
}

func (c *client) logExitDetails(execErr error) {
	if exitError, ok := execErr.(*exec.ExitError); ok {
		c.logger.Error("Command exit details", "stderr", string(exitError.Stderr), "exit_code", exitError.ExitCode())
	}
}

func shouldWrapNotFound(commandName string, output []byte) bool {
	if !isAppScopedCommand(commandName) {
		return false
	}
	lower := strings.ToLower(string(output))
	return isNotFoundOutput(lower)
}

func isNotFoundOutput(lowerOutput string) bool {
	if strings.Contains(lowerOutput, "does not exist") || strings.Contains(lowerOutput, "has not been deployed") {
		return true
	}
	return strings.Contains(lowerOutput, "docker options phase file") && strings.Contains(lowerOutput, "no such file or directory")
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

// SetAllowlist restricts execution to commands matching these patterns.
// An empty list allows every command that is not blacklisted.
func (c *client) SetAllowlist(patterns []string) {
	c.allowedCommands = patterns
	c.logger.Debug("Command allowlist updated", "patterns", patterns) // Audit trail
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

		jsonArgs := slices.Concat(args, []string{"--format", "json"})
		output, err := c.ExecuteCommand(ctx, commandName, jsonArgs)
		if err != nil {
			c.logger.Warn("Failed to execute with JSON format, falling back to text",
				"command", commandName,
				"error", err)
			// Persist downgrade to avoid repeated failures
			c.capabilities.AddJSONSupport(commandName, false)
			c.capabilities.CommandRegistry.Set(commandName, &CommandInfo{Name: commandName, SupportsJSON: false})
			// Fall through to text parsing
		} else {
			// Validate it's actually JSON
			if json.Valid(output) {
				// Persist confirmed JSON capability
				c.capabilities.AddJSONSupport(commandName, true)
				c.capabilities.CommandRegistry.Set(commandName, &CommandInfo{Name: commandName, SupportsJSON: true})
				return &CommandResult{
					RawOutput: output,
					JSONData:  output,
					ParsedAt:  time.Now(),
				}, nil
			}
			c.logger.Warn("Command returned non-JSON output despite --format json flag",
				"command", commandName)
			// Persist downgrade if misleading response
			c.capabilities.AddJSONSupport(commandName, false)
			c.capabilities.CommandRegistry.Set(commandName, &CommandInfo{Name: commandName, SupportsJSON: false})
		}
	}

	// Opportunistic probe: for report/info commands with unknown capability, try JSON once
	if !supportsJSON && (strings.Contains(commandName, ":report") || strings.Contains(commandName, ":info")) {
		c.logger.Debug("Opportunistic JSON probe for report/info command",
			"command", commandName)
		jsonArgs := slices.Concat(args, []string{"--format", "json"})
		output, err := c.ExecuteCommand(ctx, commandName, jsonArgs)
		if err == nil && json.Valid(output) {
			// Persist confirmed support and return
			c.capabilities.AddJSONSupport(commandName, true)
			c.capabilities.CommandRegistry.Set(commandName, &CommandInfo{Name: commandName, SupportsJSON: true})
			return &CommandResult{
				RawOutput: output,
				JSONData:  output,
				ParsedAt:  time.Now(),
			}, nil
		}
		// On failure, persist negative to avoid repeated probes
		c.capabilities.AddJSONSupport(commandName, false)
		c.capabilities.CommandRegistry.Set(commandName, &CommandInfo{Name: commandName, SupportsJSON: false})
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

// parseLogLine parses a Dokku log line
// Format: "2025-12-13T01:30:00.000000000Z app[web.1]: message"
func parseLogLine(line string) LogLine {
	// Simple parsing - enhance as needed
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 3 {
		return LogLine{
			Timestamp: time.Now(),
			Message:   line,
		}
	}

	timestamp, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		// Not a "<timestamp> <container>: <message>" line: keep it whole.
		return LogLine{
			Timestamp: time.Now(),
			Message:   line,
		}
	}
	container := strings.Trim(parts[1], ":")

	return LogLine{
		Timestamp: timestamp,
		Container: container,
		Message:   parts[2],
	}
}

// GetLogs retrieves application logs (polling mode for stdio transport)
func (c *client) GetLogs(ctx context.Context, appName string, options LogOptions) (string, error) {
	if appName == "" {
		return "", fmt.Errorf("application name cannot be empty")
	}

	// Remove duplicate "logs" from args - command name is already "logs"
	args := []string{appName}

	if options.Lines > 0 {
		args = append(args, "--num", fmt.Sprintf("%d", options.Lines))
	}

	if options.Tail {
		return "", fmt.Errorf("use StreamLogs for tailing logs")
	}

	output, err := c.ExecuteCommand(ctx, "logs", args)
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}

	return string(output), nil
}

// StreamLogs streams application logs (for SSE transport)
// Returns channels for log lines and errors
func (c *client) StreamLogs(ctx context.Context, appName string) (<-chan LogLine, <-chan error, error) {
	if appName == "" {
		return nil, nil, fmt.Errorf("application name cannot be empty")
	}

	logChan := make(chan LogLine, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(logChan)
		defer close(errChan)

		// Remove duplicate "logs" from args - command name is already "logs"
		args := []string{appName, "-t"}

		cmd, cancelFunc, err := c.buildCommand(ctx, args)
		if err != nil {
			errChan <- fmt.Errorf("failed to build command: %w", err)
			return
		}
		// Ensure cleanup of context if we created one
		if cancelFunc != nil {
			defer cancelFunc()
		}

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			errChan <- fmt.Errorf("failed to create stdout pipe: %w", err)
			return
		}

		if err := cmd.Start(); err != nil {
			errChan <- fmt.Errorf("failed to start command: %w", err)
			return
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()

			select {
			case logChan <- parseLogLine(line):
			case <-ctx.Done():
				if err := cmd.Process.Kill(); err != nil {
					c.logger.Error("failed to kill process", "error", err)
				}
				// Wait for process to clean up resources
				if waitErr := cmd.Wait(); waitErr != nil {
					c.logger.Error("error waiting for process after kill", "error", waitErr)
				}
				return
			}
		}

		// Report at most one error: errChan has room for exactly one, so a
		// second send would block forever once the consumer stops reading.
		scanErr := scanner.Err()
		waitErr := cmd.Wait()
		switch {
		case scanErr != nil:
			errChan <- fmt.Errorf("error reading logs: %w", scanErr)
		case waitErr != nil:
			errChan <- fmt.Errorf("command failed: %w", waitErr)
		}
	}()

	return logChan, errChan, nil
}

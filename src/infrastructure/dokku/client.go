package dokku

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

type DokkuClient interface {
	ExecuteCommand(ctx context.Context, command string, args []string) ([]byte, error)
	GetApplications(ctx context.Context) ([]string, error)
	GetApplicationInfo(ctx context.Context, appName string) (map[string]interface{}, error)
	GetApplicationConfig(ctx context.Context, appName string) (map[string]string, error)
	SetApplicationConfig(ctx context.Context, appName string, config map[string]string) error
	ScaleApplication(ctx context.Context, appName string, processType string, count int) error
	GetApplicationLogs(ctx context.Context, appName string, lines int) (string, error)
}

type ClientConfig struct {
	DokkuHost       string          `yaml:"dokku_host"`
	DokkuPort       int             `yaml:"dokku_port"`
	DokkuUser       string          `yaml:"dokku_user"`
	DokkuPath       string          `yaml:"dokku_path"`
	SSHKeyPath      string          `yaml:"ssh_key_path"`
	CommandTimeout  time.Duration   `yaml:"command_timeout"`
	AllowedCommands map[string]bool `yaml:"allowed_commands"`
}

func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		DokkuHost:      "dokku.me",
		DokkuPort:      22,
		DokkuUser:      "dokku",
		DokkuPath:      "/usr/bin/dokku",
		SSHKeyPath:     "",
		CommandTimeout: 30 * time.Second,
		AllowedCommands: map[string]bool{
			"apps:list":    true,
			"apps:info":    true,
			"apps:create":  true,
			"apps:exists":  true,
			"config:get":   true,
			"config:set":   true,
			"config:show":  true,
			"domains:add":  true,
			"domains:list": true,
			"ps:scale":     true,
			"logs":         true,
			"events":       true,
			"git:report":   true,
		},
	}
}

type client struct {
	config      *ClientConfig
	logger      *slog.Logger
	authService *SSHAuthService
}

func NewDokkuClient(config *ClientConfig, logger *slog.Logger) DokkuClient {
	if config == nil {
		config = DefaultClientConfig()
	}
	return &client{
		config:      config,
		logger:      logger,
		authService: NewSSHAuthService(logger),
	}
}

func (c *client) ExecuteCommand(ctx context.Context, command string, args []string) ([]byte, error) {
	if !c.config.AllowedCommands[command] {
		return nil, fmt.Errorf("command not allowed: %s", command)
	}

	for i, arg := range args {
		if err := c.validateCommandArgument(arg); err != nil {
			return nil, fmt.Errorf("invalid argument %d: %w", i, err)
		}
	}

	cmdCtx, cancel := context.WithTimeout(ctx, c.config.CommandTimeout)
	defer cancel()

	dokkuArgs := append([]string{c.config.DokkuPath, command}, args...)
	sshArgs := []string{
		"-o", "LogLevel=QUIET",
		"-p", fmt.Sprintf("%d", c.config.DokkuPort),
		"-t",
		fmt.Sprintf("%s@%s", c.config.DokkuUser, c.config.DokkuHost),
		"--",
	}
	sshArgs = append(sshArgs, dokkuArgs...)

	authMethod := c.authService.DetermineAuthMethod(c.config.SSHKeyPath)

	sshArgs = c.authService.PrepareSSHArgs(authMethod, sshArgs)

	cmd := exec.CommandContext(cmdCtx, "ssh", sshArgs...)

	// Security: minimal environment
	baseEnv := []string{
		"PATH=/usr/bin:/bin",
		fmt.Sprintf("DOKKU_HOST=%s", c.config.DokkuHost),
		fmt.Sprintf("DOKKU_PORT=%d", c.config.DokkuPort),
	}

	cmd.Env = c.authService.PrepareEnvironment(authMethod, baseEnv)

	c.logger.Debug("Executing Dokku command via SSH",
		"command", command,
		"args", args,
		"ssh_target", fmt.Sprintf("%s@%s:%d", c.config.DokkuUser, c.config.DokkuHost, c.config.DokkuPort),
		"auth_method", authMethod.Description,
		"ssh_args", sshArgs,
		"dokku_args", dokkuArgs)

	output, err := cmd.Output()
	if err != nil {
		c.logger.Error("Failed to execute Dokku command",
			"error", err,
			"command", command,
			"args", args,
			"auth_method", authMethod.Description,
			"ssh_args", sshArgs)
		return nil, fmt.Errorf("failed to execute Dokku command %s: %w", command, err)
	}

	c.logger.Debug("Dokku command executed successfully",
		"command", command,
		"output_size", len(output),
		"output_preview", func() string {
			if len(output) > 200 {
				return string(output[:200])
			}
			return string(output)
		}(),
		"auth_method", authMethod.Description)

	return output, nil
}

func (c *client) GetApplications(ctx context.Context) ([]string, error) {
	output, err := c.ExecuteCommand(ctx, "apps:list", []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to get applications: %w", err)
	}

	// Log de debug pour voir la sortie brute
	c.logger.Debug("Sortie brute de dokku apps:list",
		"output", string(output),
		"output_len", len(output))

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var apps []string

	// Log de debug pour voir chaque ligne
	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		c.logger.Debug("Traitement ligne",
			"index", i,
			"line_raw", line,
			"line_trimmed", trimmedLine,
			"starts_with_equals", strings.HasPrefix(trimmedLine, "===="),
			"is_empty", trimmedLine == "")

		if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "====") {
			apps = append(apps, trimmedLine)
			c.logger.Debug("Application trouvée", "app_name", trimmedLine)
		}
	}

	c.logger.Debug("Applications retrieved",
		"count", len(apps),
		"apps", apps)

	return apps, nil
}

func (c *client) parseOutputLines(output []byte, separator string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.Contains(line, separator) {
			parts := strings.SplitN(line, separator, 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				result[key] = value
			}
		}
	}

	return result
}

func (c *client) GetApplicationInfo(ctx context.Context, appName string) (map[string]interface{}, error) {
	output, err := c.ExecuteCommand(ctx, "apps:info", []string{appName})
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			c.logger.Debug("apps:info returned exit status 1 - application probably not deployed",
				"app_name", appName,
				"suggestion", "application exists but has no detailed information available")
		}
		return nil, fmt.Errorf("failed to get application info %s: %w", appName, err)
	}

	// Use the common function to parse key:value pairs
	stringMap := c.parseOutputLines(output, ":")

	info := make(map[string]any)
	for key, value := range stringMap {
		info[key] = value
	}

	return info, nil
}

func (c *client) GetApplicationConfig(ctx context.Context, appName string) (map[string]string, error) {
	output, err := c.ExecuteCommand(ctx, "config:show", []string{appName})
	if err != nil {
		return nil, fmt.Errorf("failed to get application config %s: %w", appName, err)
	}

	config := c.parseOutputLines(output, "=")
	return config, nil
}

func (c *client) SetApplicationConfig(ctx context.Context, appName string, config map[string]string) error {
	var args []string
	args = append(args, appName)

	for key, value := range config {
		args = append(args, fmt.Sprintf("%s=%s", key, value))
	}

	_, err := c.ExecuteCommand(ctx, "config:set", args)
	if err != nil {
		return fmt.Errorf("failed to set application config %s: %w", appName, err)
	}

	return nil
}

func (c *client) ScaleApplication(ctx context.Context, appName string, processType string, count int) error {
	scaleArg := fmt.Sprintf("%s=%d", processType, count)
	_, err := c.ExecuteCommand(ctx, "ps:scale", []string{appName, scaleArg})
	if err != nil {
		return fmt.Errorf("failed to scale application %s: %w", appName, err)
	}

	return nil
}

func (c *client) GetApplicationLogs(ctx context.Context, appName string, lines int) (string, error) {
	args := []string{appName}
	if lines > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", lines))
	}

	output, err := c.ExecuteCommand(ctx, "logs", args)
	if err != nil {
		return "", fmt.Errorf("failed to get application logs %s: %w", appName, err)
	}

	return string(output), nil
}

// Prevents command injection attacks
func (c *client) validateCommandArgument(arg string) error {
	dangerous := []string{";", "|", "&", "$", "`", "(", ")", "{", "}", "[", "]", "<", ">", "\n", "\r"}
	for _, char := range dangerous {
		if strings.Contains(arg, char) {
			return fmt.Errorf("dangerous character detected: %s", char)
		}
	}

	if len(arg) > 255 {
		return fmt.Errorf("argument too long (max 255 characters)")
	}

	return nil
}

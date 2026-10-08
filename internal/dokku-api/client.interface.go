package dokkuApi

import "context"

// CommandExecutor defines the core command execution capability
type CommandExecutor interface {
	ExecuteCommand(ctx context.Context, command string, args []string) ([]byte, error)
}

// StdinExecutor runs commands that read their payload from stdin.
type StdinExecutor interface {
	ExecuteCommandWithStdin(ctx context.Context, command string, args []string, stdin string) ([]byte, error)
}

// LogStreamer follows application logs until the context is cancelled.
type LogStreamer interface {
	StreamLogs(ctx context.Context, appName string) (<-chan LogLine, <-chan error, error)
}

// CommandParser defines parsing capabilities for different output formats
type CommandParser interface {
	GetKeyValueOutput(ctx context.Context, command string, args []string, separator string) (map[string]string, error)
	GetListOutput(ctx context.Context, command string, args []string) ([]string, error)
	GetTableOutput(ctx context.Context, command string, args []string, skipHeaders bool) ([]map[string]string, error)
}

// StructuredExecutor combines execution with structured parsing
type StructuredExecutor interface {
	ExecuteStructured(ctx context.Context, spec CommandSpec) (*CommandResult, error)
	ExecuteWithAutoFormat(ctx context.Context, commandName string, args []string) (*CommandResult, error)
}

// CapabilityManager defines capability discovery and management
type CapabilityManager interface {
	DiscoverCapabilities(ctx context.Context) error
	GetCapabilities() *DokkuCapabilities
}

// SSHManager defines SSH connection management
type SSHManager interface {
	GetSSHConnectionManager() *SSHConnectionManager
}

// CommandFilter defines command filtering/security capabilities
type CommandFilter interface {
	SetBlacklist(commands []string)
	SetAllowlist(patterns []string)
	ValidateCommand(command string, args []string) error
}

// DokkuClient combines all Dokku-specific capabilities
// This is the "convenience interface" that most consumers will use
type DokkuClient interface {
	CommandExecutor
	StdinExecutor
	LogStreamer
	CommandParser
	StructuredExecutor
	CapabilityManager
	SSHManager
	CommandFilter
}

// For consumers that only need basic execution (better testability)
type DokkuExecutor interface {
	CommandExecutor
}

// For consumers that need parsing capabilities
type DokkuParser interface {
	CommandExecutor
	CommandParser
}

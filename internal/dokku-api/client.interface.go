package dokkuApi

import "context"

// CommandExecutor defines the core command execution capability
type CommandExecutor interface {
	ExecuteCommand(ctx context.Context, command string, args []string) ([]byte, error)
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
}

// SSHManager defines SSH connection management
type SSHManager interface {
	GetSSHConnectionManager() *SSHConnectionManager
}

// CommandFilter defines command filtering/security capabilities
type CommandFilter interface {
	SetBlacklist(commands []string)
}

// DokkuClient combines all Dokku-specific capabilities
// This is the "convenience interface" that most consumers will use
type DokkuClient interface {
	CommandExecutor
	CommandParser
	StructuredExecutor
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

package dokkuApi

import (
	"log/slog"
	"time"
)

// OutputFormat represents different output parsing strategies
type OutputFormat string

const (
	OutputFormatKeyValue OutputFormat = "key_value" // key: value or key=value
	OutputFormatList     OutputFormat = "list"      // line-separated items
	OutputFormatTable    OutputFormat = "table"     // column-based data
	OutputFormatJSON     OutputFormat = "json"      // JSON output
	OutputFormatRaw      OutputFormat = "raw"       // raw string output
)

// CommandResult represents the result of a Dokku command with parsed data
type CommandResult struct {
	RawOutput    []byte
	KeyValueData map[string]string
	ListData     []string
	TableData    []map[string]string
	JSONData     interface{}
	ParsedAt     time.Time
}

// CommandSpec defines how to execute and parse a command
type CommandSpec struct {
	Command      string
	Args         []string
	OutputFormat OutputFormat
	Separator    string // for key-value parsing (e.g., ":", "=")
	SkipHeaders  bool   // for table parsing
	FilterEmpty  bool   // skip empty lines
}

type ClientConfig struct {
	DokkuHost      string        `yaml:"dokku_host"`
	DokkuPort      int           `yaml:"dokku_port"`
	DokkuUser      string        `yaml:"dokku_user"`
	DokkuPath      string        `yaml:"dokku_path"`
	SSHKeyPath     string        `yaml:"ssh_key_path"`
	CommandTimeout time.Duration `yaml:"command_timeout"`
	Cache          *CacheConfig  `yaml:"cache"`
}

func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		DokkuHost:      "pro.dokku.com",
		DokkuPort:      22,
		DokkuUser:      "dokku",
		DokkuPath:      "/usr/bin/dokku",
		SSHKeyPath:     "",
		CommandTimeout: 30 * time.Second,
		Cache:          DefaultCacheConfig(),
	}
}

type client struct {
	config              *ClientConfig
	logger              *slog.Logger
	sshConnManager      *SSHConnectionManager
	blacklistedCommands []string

	// Optional caching - managed by cache manager
	cacheManager *CommandCacheManager
}

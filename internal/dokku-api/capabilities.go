package dokkuApi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// DokkuCapabilities represents the capabilities and version information of a Dokku installation
type DokkuCapabilities struct {
	Version         string           `json:"version"`
	Plugins         []string         `json:"plugins"`
	CommandRegistry *CommandRegistry `json:"-"`
	JSONSupport     map[string]bool  `json:"json_support"`
	mu              sync.RWMutex     `json:"-"`
	lastUpdated     time.Time        `json:"-"`
}

// CommandRegistry tracks which commands are available and their characteristics
type CommandRegistry struct {
	commands map[string]*CommandInfo
	mu       sync.RWMutex
}

// CommandInfo contains metadata about a specific Dokku command
type CommandInfo struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	SupportsJSON bool     `json:"supports_json"`
	Args         []string `json:"args,omitempty"`
}

// NewDokkuCapabilities creates a new capabilities instance
func NewDokkuCapabilities() *DokkuCapabilities {
	return &DokkuCapabilities{
		Version:         "unknown",
		Plugins:         []string{},
		CommandRegistry: NewCommandRegistry(),
		JSONSupport:     make(map[string]bool),
		lastUpdated:     time.Now(),
	}
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]*CommandInfo),
	}
}

// Get retrieves command information by name
func (cr *CommandRegistry) Get(commandName string) *CommandInfo {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.commands[commandName]
}

// Set stores command information
func (cr *CommandRegistry) Set(commandName string, info *CommandInfo) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	cr.commands[commandName] = info
}

// List returns all registered commands
func (cr *CommandRegistry) List() []string {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	commands := make([]string, 0, len(cr.commands))
	for name := range cr.commands {
		commands = append(commands, name)
	}
	return commands
}

// SupportsJSON checks if a command supports JSON output
func (dc *DokkuCapabilities) SupportsJSON(commandName string, version string) bool {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	// Check if we have explicit information about this command
	if supported, exists := dc.JSONSupport[commandName]; exists {
		return supported
	}

	// Fall back to version-based heuristics
	return dc.supportsJSONByVersion(version)
}

// supportsJSONByVersion determines JSON support based on Dokku version
func (dc *DokkuCapabilities) supportsJSONByVersion(version string) bool {
	// Basic version parsing - this is a simplified approach
	// In practice, you might want more sophisticated version comparison
	if version == "unknown" {
		return false
	}

	// Assume JSON support is available in modern Dokku versions
	// This is a heuristic and should be refined based on actual testing
	return true
}

// UpdateVersion updates the Dokku version
func (dc *DokkuCapabilities) UpdateVersion(version string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.Version = version
	dc.lastUpdated = time.Now()
}

// UpdatePlugins updates the list of available plugins
func (dc *DokkuCapabilities) UpdatePlugins(plugins []string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.Plugins = plugins
	dc.lastUpdated = time.Now()
}

// AddJSONSupport marks a command as supporting JSON output
func (dc *DokkuCapabilities) AddJSONSupport(commandName string, supported bool) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.JSONSupport[commandName] = supported
}

// IsStale checks if the capabilities data is stale
func (dc *DokkuCapabilities) IsStale(maxAge time.Duration) bool {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return time.Since(dc.lastUpdated) > maxAge
}

// Clone creates a deep copy of the capabilities
func (dc *DokkuCapabilities) Clone() *DokkuCapabilities {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	clone := &DokkuCapabilities{
		Version:         dc.Version,
		Plugins:         make([]string, len(dc.Plugins)),
		CommandRegistry: NewCommandRegistry(),
		JSONSupport:     make(map[string]bool),
		lastUpdated:     dc.lastUpdated,
	}

	copy(clone.Plugins, dc.Plugins)

	// Copy command registry
	for name, info := range dc.CommandRegistry.commands {
		clone.CommandRegistry.Set(name, &CommandInfo{
			Name:         info.Name,
			Description:  info.Description,
			SupportsJSON: info.SupportsJSON,
			Args:         make([]string, len(info.Args)),
		})
		copy(clone.CommandRegistry.commands[name].Args, info.Args)
	}

	// Copy JSON support map
	for cmd, supported := range dc.JSONSupport {
		clone.JSONSupport[cmd] = supported
	}

	return clone
}

// String returns a string representation of the capabilities
func (dc *DokkuCapabilities) String() string {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	return fmt.Sprintf("DokkuCapabilities{Version: %s, Plugins: %d, Commands: %d, JSONSupport: %d}",
		dc.Version, len(dc.Plugins), len(dc.CommandRegistry.commands), len(dc.JSONSupport))
}

// DiscoverCapabilities discovers the capabilities of a Dokku installation
func (c *client) DiscoverCapabilities(ctx context.Context) error {
	c.logger.Debug("Starting Dokku capabilities discovery")

	// Discover version
	if err := c.discoverVersion(ctx); err != nil {
		c.logger.Warn("Failed to discover Dokku version", "error", err)
	}

	// Discover plugins
	if err := c.discoverPlugins(ctx); err != nil {
		c.logger.Warn("Failed to discover Dokku plugins", "error", err)
	}

	// Discover command capabilities
	if err := c.discoverCommandCapabilities(ctx); err != nil {
		c.logger.Warn("Failed to discover command capabilities", "error", err)
	}

	c.logger.Debug("Dokku capabilities discovery completed",
		"version", c.capabilities.Version,
		"plugins_count", len(c.capabilities.Plugins),
		"commands_count", len(c.capabilities.CommandRegistry.List()))

	return nil
}

// GetCapabilities returns the current capabilities
func (c *client) GetCapabilities() *DokkuCapabilities {
	return c.capabilities.Clone()
}

// discoverVersion discovers the Dokku version
func (c *client) discoverVersion(ctx context.Context) error {
	output, err := c.executeCommandDirect(ctx, "version", []string{})
	if err != nil {
		return fmt.Errorf("failed to get Dokku version: %w", err)
	}

	version := strings.TrimSpace(string(output))
	c.capabilities.UpdateVersion(version)

	c.logger.Debug("Discovered Dokku version", "version", version)
	return nil
}

// discoverPlugins discovers available Dokku plugins
func (c *client) discoverPlugins(ctx context.Context) error {
	output, err := c.executeCommandDirect(ctx, "plugin:list", []string{})
	if err != nil {
		return fmt.Errorf("failed to get Dokku plugins: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	var plugins []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "plug") {
			// Extract plugin name (first word)
			parts := strings.Fields(line)
			if len(parts) > 0 {
				plugins = append(plugins, parts[0])
			}
		}
	}

	c.capabilities.UpdatePlugins(plugins)

	c.logger.Debug("Discovered Dokku plugins", "count", len(plugins))
	return nil
}

// discoverCommandCapabilities discovers capabilities of specific commands
func (c *client) discoverCommandCapabilities(ctx context.Context) error {
	// Test common commands for JSON support
	commonCommands := []string{
		"apps:list",
		"apps:report",
		"config:show",
		"domains:list",
		"domains:report",
	}

	for _, cmd := range commonCommands {
		// Try to execute with --format json
		output, err := c.executeCommandDirect(ctx, cmd, []string{"--format", "json"})
		if err != nil {
			// Command doesn't support JSON or failed
			c.capabilities.AddJSONSupport(cmd, false)
			continue
		}

		// Try to parse as JSON
		if json.Valid(output) {
			c.capabilities.AddJSONSupport(cmd, true)
			c.logger.Debug("Command supports JSON", "command", cmd)
		} else {
			c.capabilities.AddJSONSupport(cmd, false)
		}
	}

	return nil
}

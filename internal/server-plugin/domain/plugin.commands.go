package domain

// PluginCommand represents allowed Dokku commands for plugin discovery
type PluginCommand string

const (
	CommandPluginList PluginCommand = "plugin:list"
)

// IsValid checks if the command is a valid plugin command
func (c PluginCommand) IsValid() bool {
	switch c {
	case CommandPluginList:
		return true
	default:
		return false
	}
}

// String returns the string representation of the command
func (c PluginCommand) String() string {
	return string(c)
}

// GetAllowedCommands returns all allowed plugin commands
func GetAllowedPluginCommands() []PluginCommand {
	return []PluginCommand{
		CommandPluginList,
	}
}

package app

// ApplicationCommand represents allowed Dokku commands for the application plugin
type ApplicationCommand string

const (
	// Application management commands
	CommandAppsList    ApplicationCommand = "apps:list"
	CommandAppsInfo    ApplicationCommand = "apps:info"
	CommandAppsCreate  ApplicationCommand = "apps:create"
	CommandAppsDestroy ApplicationCommand = "apps:destroy"
	CommandAppsExists  ApplicationCommand = "apps:exists"
	CommandAppsReport  ApplicationCommand = "apps:report"

	// Configuration commands
	CommandConfigShow ApplicationCommand = "config:show"
	CommandConfigSet  ApplicationCommand = "config:set"

	// Process management commands
	CommandPsScale   ApplicationCommand = "ps:scale"
	CommandPsReport  ApplicationCommand = "ps:report"
	CommandPsRestart ApplicationCommand = "ps:restart"
	CommandPsStop    ApplicationCommand = "ps:stop"
	CommandPsStart   ApplicationCommand = "ps:start"

	// Logging commands
	CommandLogs       ApplicationCommand = "logs"
	CommandLogsFailed ApplicationCommand = "logs:failed"
)

// IsValid checks if the command is a valid application command
func (c ApplicationCommand) IsValid() bool {
	switch c {
	case CommandAppsList, CommandAppsInfo, CommandAppsCreate, CommandAppsDestroy,
		CommandAppsExists, CommandAppsReport, CommandConfigShow, CommandConfigSet,
		CommandPsScale, CommandPsReport, CommandPsRestart, CommandPsStop, CommandPsStart,
		CommandLogs, CommandLogsFailed:
		return true
	default:
		return false
	}
}

// String returns the string representation of the command
func (c ApplicationCommand) String() string {
	return string(c)
}

// GetAllowedCommands returns all allowed application commands
func GetAllowedCommands() []ApplicationCommand {
	return []ApplicationCommand{
		CommandAppsList,
		CommandAppsInfo,
		CommandAppsCreate,
		CommandAppsDestroy,
		CommandAppsExists,
		CommandAppsReport,
		CommandConfigShow,
		CommandConfigSet,
		CommandPsScale,
		CommandPsReport,
		CommandPsRestart,
		CommandPsStop,
		CommandPsStart,
		CommandLogs,
		CommandLogsFailed,
	}
}

package domain

// DeploymentCommand represents allowed Dokku commands for the deployment plugin
type DeploymentCommand string

const (
	// Buildpack commands
	CommandBuildpacksSet DeploymentCommand = "buildpacks:set"

	// Git commands
	CommandGitSync DeploymentCommand = "git:sync"

	// Process commands
	CommandPsRebuild DeploymentCommand = "ps:rebuild"

	// Event commands
	CommandEvents DeploymentCommand = "events"
)

// IsValid checks if the command is a valid deployment command
func (c DeploymentCommand) IsValid() bool {
	switch c {
	case CommandBuildpacksSet,
		CommandGitSync, CommandPsRebuild, CommandEvents:
		return true
	default:
		return false
	}
}

// String returns the string representation of the command
func (c DeploymentCommand) String() string {
	return string(c)
}

// GetAllowedCommands returns all allowed deployment commands
func GetAllowedDeploymentCommands() []DeploymentCommand {
	return []DeploymentCommand{
		CommandBuildpacksSet,
		CommandGitSync,
		CommandPsRebuild,
		CommandEvents,
	}
}

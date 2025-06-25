package domain

// DomainCommand represents allowed Dokku commands for the domain plugin
type DomainCommand string

const (
	CommandDomainsReport       DomainCommand = "domains:report"
	CommandDomainsAddGlobal    DomainCommand = "domains:add-global"
	CommandDomainsRemoveGlobal DomainCommand = "domains:remove-global"
	CommandDomainsSetGlobal    DomainCommand = "domains:set-global"
	CommandDomainsClearGlobal  DomainCommand = "domains:clear-global"
	CommandLetsEncryptSet      DomainCommand = "letsencrypt:set"
)

// IsValid checks if the command is a valid domain command
func (c DomainCommand) IsValid() bool {
	switch c {
	case CommandDomainsReport, CommandDomainsAddGlobal, CommandDomainsRemoveGlobal,
		CommandDomainsSetGlobal, CommandDomainsClearGlobal, CommandLetsEncryptSet:
		return true
	default:
		return false
	}
}

// String returns the string representation of the command
func (c DomainCommand) String() string {
	return string(c)
}

// GetAllowedCommands returns all allowed domain commands
func GetAllowedCommands() []DomainCommand {
	return []DomainCommand{
		CommandDomainsReport,
		CommandDomainsAddGlobal,
		CommandDomainsRemoveGlobal,
		CommandDomainsSetGlobal,
		CommandDomainsClearGlobal,
		CommandLetsEncryptSet,
	}
}

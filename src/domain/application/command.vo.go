package application

import (
	"fmt"
	"regexp"
	"strings"
)

// DokkuCommand représente une commande Dokku validée et sécurisée
type DokkuCommand struct {
	name string
	args []string
}

var (
	// Commandes autorisées par défaut
	AllowedCommands = map[string]bool{
		"apps:list":         true,
		"apps:info":         true,
		"apps:create":       true,
		"apps:destroy":      true,
		"apps:exists":       true,
		"apps:report":       true,
		"config:get":        true,
		"config:set":        true,
		"config:unset":      true,
		"config:show":       true,
		"domains:add":       true,
		"domains:remove":    true,
		"domains:list":      true,
		"domains:report":    true,
		"ps:scale":          true,
		"ps:restart":        true,
		"ps:start":          true,
		"ps:stop":           true,
		"ps:report":         true,
		"logs":              true,
		"events":            true,
		"git:report":        true,
		"git:sync":          true,
		"buildpacks:set":    true,
		"buildpacks:clear":  true,
		"buildpacks:report": true,
		"service:create":    true,
		"service:destroy":   true,
		"service:link":      true,
		"service:unlink":    true,
		"service:list":      true,
	}

	// Pattern pour valider les arguments
	safeArgPattern = regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)
	appNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`)
)

// NewDokkuCommand crée une nouvelle commande Dokku avec validation
func NewDokkuCommand(name string, args []string) (*DokkuCommand, error) {
	name = strings.TrimSpace(name)

	if err := validateCommand(name, args); err != nil {
		return nil, fmt.Errorf("commande Dokku invalide: %w", err)
	}

	// Copier les arguments pour éviter les modifications externes
	safeArgs := make([]string, len(args))
	copy(safeArgs, args)

	return &DokkuCommand{
		name: name,
		args: safeArgs,
	}, nil
}

// MustNewDokkuCommand crée une commande en paniquant en cas d'erreur
func MustNewDokkuCommand(name string, args []string) *DokkuCommand {
	cmd, err := NewDokkuCommand(name, args)
	if err != nil {
		panic(err)
	}
	return cmd
}

// Name retourne le nom de la commande
func (c *DokkuCommand) Name() string {
	return c.name
}

// Args retourne les arguments de la commande (copie)
func (c *DokkuCommand) Args() []string {
	args := make([]string, len(c.args))
	copy(args, c.args)
	return args
}

// String retourne la représentation string de la commande
func (c *DokkuCommand) String() string {
	if len(c.args) == 0 {
		return c.name
	}
	return fmt.Sprintf("%s %s", c.name, strings.Join(c.args, " "))
}

// IsAppCommand vérifie si c'est une commande qui agit sur une app
func (c *DokkuCommand) IsAppCommand() bool {
	appCommands := []string{
		"apps:info", "apps:destroy", "apps:exists", "apps:report",
		"config:get", "config:set", "config:unset", "config:show",
		"domains:add", "domains:remove", "domains:list", "domains:report",
		"ps:scale", "ps:restart", "ps:start", "ps:stop", "ps:report",
		"logs", "git:report", "git:sync",
		"buildpacks:set", "buildpacks:clear", "buildpacks:report",
	}

	for _, appCmd := range appCommands {
		if c.name == appCmd {
			return true
		}
	}
	return false
}

// GetAppName retourne le nom de l'app si c'est une commande d'app
func (c *DokkuCommand) GetAppName() string {
	if c.IsAppCommand() && len(c.args) > 0 {
		return c.args[0]
	}
	return ""
}

// IsReadOnlyCommand vérifie si c'est une commande en lecture seule
func (c *DokkuCommand) IsReadOnlyCommand() bool {
	readOnlyCommands := []string{
		"apps:list", "apps:info", "apps:exists", "apps:report",
		"config:get", "config:show",
		"domains:list", "domains:report",
		"ps:report", "logs", "events",
		"git:report", "buildpacks:report",
		"service:list",
	}

	for _, readCmd := range readOnlyCommands {
		if c.name == readCmd {
			return true
		}
	}
	return false
}

// Equal compare deux commandes
func (c *DokkuCommand) Equal(other *DokkuCommand) bool {
	if other == nil {
		return false
	}

	if c.name != other.name {
		return false
	}

	if len(c.args) != len(other.args) {
		return false
	}

	for i, arg := range c.args {
		if arg != other.args[i] {
			return false
		}
	}

	return true
}

// validateCommand valide une commande et ses arguments
func validateCommand(name string, args []string) error {
	if name == "" {
		return fmt.Errorf("le nom de la commande ne peut pas être vide")
	}

	// Vérifier si la commande est autorisée
	if !AllowedCommands[name] {
		return fmt.Errorf("commande non autorisée: %s", name)
	}

	// Valider les arguments
	for i, arg := range args {
		if err := validateArgument(arg, i, name); err != nil {
			return fmt.Errorf("argument %d invalide: %w", i, err)
		}
	}

	// Validations spécifiques par type de commande
	return validateCommandSpecific(name, args)
}

// validateArgument valide un argument spécifique
func validateArgument(arg string, index int, command string) error {
	if arg == "" {
		return fmt.Errorf("l'argument ne peut pas être vide")
	}

	if len(arg) > 200 {
		return fmt.Errorf("l'argument est trop long (max 200 caractères)")
	}

	// Vérifications de sécurité
	if strings.Contains(arg, "..") {
		return fmt.Errorf("l'argument ne peut pas contenir '..'")
	}

	if strings.ContainsAny(arg, ";|&$`(){}[]<>\"'\\") {
		return fmt.Errorf("l'argument contient des caractères dangereux")
	}

	// Validation du nom d'app (premier argument pour les commandes d'app)
	if index == 0 && isAppNameArgument(command) {
		if !appNamePattern.MatchString(arg) {
			return fmt.Errorf("nom d'application invalide: %s", arg)
		}
	}

	return nil
}

// validateCommandSpecific valide les commandes avec des règles spécifiques
func validateCommandSpecific(name string, args []string) error {
	switch name {
	case "ps:scale":
		if len(args) < 2 {
			return fmt.Errorf("ps:scale nécessite au moins 2 arguments")
		}
		// Validation du format process-type=count
		for i := 1; i < len(args); i++ {
			if !strings.Contains(args[i], "=") {
				return fmt.Errorf("format de scaling invalide: %s", args[i])
			}
		}

	case "config:set":
		if len(args) < 2 {
			return fmt.Errorf("config:set nécessite au moins 2 arguments")
		}
		// Validation du format KEY=VALUE
		for i := 1; i < len(args); i++ {
			if !strings.Contains(args[i], "=") {
				return fmt.Errorf("format de configuration invalide: %s", args[i])
			}
		}

	case "domains:add":
		if len(args) != 2 {
			return fmt.Errorf("domains:add nécessite exactement 2 arguments")
		}
		// Le deuxième argument doit être un domaine valide
		if _, err := NewDomain(args[1]); err != nil {
			return fmt.Errorf("domaine invalide: %w", err)
		}
	}

	return nil
}

// isAppNameArgument vérifie si le premier argument est un nom d'app
func isAppNameArgument(command string) bool {
	appCommands := []string{
		"apps:info", "apps:destroy", "apps:exists", "apps:report",
		"config:get", "config:set", "config:unset", "config:show",
		"domains:add", "domains:remove", "domains:list", "domains:report",
		"ps:scale", "ps:restart", "ps:start", "ps:stop", "ps:report",
		"logs", "git:report", "git:sync",
		"buildpacks:set", "buildpacks:clear", "buildpacks:report",
	}

	for _, appCmd := range appCommands {
		if command == appCmd {
			return true
		}
	}
	return false
}

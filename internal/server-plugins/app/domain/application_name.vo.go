package app

import (
	"fmt"
	"regexp"
	"strings"
)

// ApplicationName represents a valid Dokku application name
type ApplicationName struct {
	value string
}

var (
	// Pattern to validate a Dokku application name
	// Must respect DNS and Dokku conventions
	applicationNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
)

// NewApplicationName creates a new application name with validation
func NewApplicationName(name string) (*ApplicationName, error) {
	name = strings.TrimSpace(strings.ToLower(name))

	if err := validateApplicationName(name); err != nil {
		return nil, fmt.Errorf("invalid application name: %w", err)
	}

	return &ApplicationName{value: name}, nil
}

// MustNewApplicationName creates an application name, panicking on error
func MustNewApplicationName(name string) *ApplicationName {
	appName, err := NewApplicationName(name)
	if err != nil {
		panic(fmt.Sprintf("cannot create application name %s: %v", name, err))
	}
	return appName
}

// Value returns the value of the application name
func (an *ApplicationName) Value() string {
	return an.value
}

// String implements fmt.Stringer
func (an *ApplicationName) String() string {
	return an.value
}

// Equal compares two application names
func (an *ApplicationName) Equal(other *ApplicationName) bool {
	if other == nil {
		return false
	}
	return an.value == other.value
}

// IsReserved checks if the name is a reserved name by Dokku
func (an *ApplicationName) IsReserved() bool {
	reservedNames := []string{
		"dokku", "tls", "app", "plugin", "plugins", "config", "logs",
		"ps", "run", "shell", "enter", "backup", "restore", "certs",
		"domains", "git", "storage", "network", "proxy", "apps",
		"service", "services", "builder", "scheduler", "registry",
	}

	for _, reserved := range reservedNames {
		if an.value == reserved {
			return true
		}
	}
	return false
}

// validateApplicationName validates an application name according to Dokku rules
func validateApplicationName(name string) error {
	if name == "" {
		return fmt.Errorf("application name cannot be empty")
	}

	if len(name) > 63 {
		return fmt.Errorf("application name cannot exceed 63 characters")
	}

	if len(name) < 1 {
		return fmt.Errorf("application name must contain at least 1 character")
	}

	// Check DNS pattern
	if !applicationNamePattern.MatchString(name) {
		return fmt.Errorf("application name must respect DNS format (lowercase letters, numbers, hyphens)")
	}

	// Cannot start or end with a hyphen
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("application name cannot start or end with a hyphen")
	}

	// Check reserved names
	reservedNames := []string{
		"dokku", "tls", "app", "plugin", "plugins", "config", "logs",
		"ps", "run", "shell", "enter", "backup", "restore", "certs",
		"domains", "git", "storage", "network", "proxy", "apps",
		"service", "services", "builder", "scheduler", "registry",
	}

	for _, reserved := range reservedNames {
		if name == reserved {
			return fmt.Errorf("name '%s' is reserved by Dokku", name)
		}
	}

	return nil
}

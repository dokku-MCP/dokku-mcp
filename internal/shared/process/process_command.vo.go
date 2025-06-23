package process

import (
	"fmt"
	"strings"
)

// ProcessCommand represents a command for a process as a value object.
type ProcessCommand struct {
	value string
}

// NewProcessCommand creates a new ProcessCommand value object.
// It returns an error if the command is empty or too long.
func NewProcessCommand(value string) (*ProcessCommand, error) {
	trimmedValue := strings.TrimSpace(value)
	if trimmedValue == "" {
		return nil, fmt.Errorf("process command cannot be empty")
	}
	// Add any other validation rules here, e.g. length
	return &ProcessCommand{value: trimmedValue}, nil
}

// Value returns the string representation of the process command.
func (c *ProcessCommand) Value() string {
	return c.value
}

// Equal checks if two ProcessCommand objects are equal.
func (c *ProcessCommand) Equal(other *ProcessCommand) bool {
	if other == nil {
		return false
	}
	return c.value == other.value
}

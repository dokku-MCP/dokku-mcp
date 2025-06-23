package shared

import (
	"fmt"
	"regexp"
)

var (
	// EnvVarKeyRegex defines the validation for an environment variable key.
	// It must be a valid C identifier (letters, numbers, and underscore).
	EnvVarKeyRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// EnvVarKey represents an environment variable key as a value object.
type EnvVarKey struct {
	value string
}

// NewEnvVarKey creates a new EnvVarKey value object.
// It returns an error if the key is invalid.
func NewEnvVarKey(value string) (*EnvVarKey, error) {
	if !EnvVarKeyRegex.MatchString(value) {
		return nil, fmt.Errorf("invalid environment variable key format: %s", value)
	}
	return &EnvVarKey{value: value}, nil
}

// Value returns the string representation of the environment variable key.
func (k *EnvVarKey) Value() string {
	return k.value
}

// Equal checks if two EnvVarKey objects are equal.
func (k *EnvVarKey) Equal(other *EnvVarKey) bool {
	if other == nil {
		return false
	}
	return k.value == other.value
}

// EnvVarValue represents an environment variable value as a value object.
type EnvVarValue struct {
	value string
}

// NewEnvVarValue creates a new EnvVarValue value object.
func NewEnvVarValue(value string) *EnvVarValue {
	return &EnvVarValue{value: value}
}

// Value returns the string representation of the environment variable value.
func (v *EnvVarValue) Value() string {
	return v.value
}

// Equal checks if two EnvVarValue objects are equal.
func (v *EnvVarValue) Equal(other *EnvVarValue) bool {
	if other == nil {
		return false
	}
	return v.value == other.value
}

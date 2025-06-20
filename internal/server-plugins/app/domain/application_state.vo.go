package app

import (
	"fmt"
	"slices"
)

// StateValue represents the possible values of an application state
type StateValue string

const (
	StateExists  StateValue = "exists"  // Application exists in Dokku but status unknown
	StateRunning StateValue = "running" // Application is running (has active processes)
	StateStopped StateValue = "stopped" // Application exists but is not running
	StateError   StateValue = "error"   // Application is in an error state
)

// ApplicationState represents the state of an application
type ApplicationState struct {
	value StateValue
}

// NewApplicationState creates a new application state
func NewApplicationState(state StateValue) (*ApplicationState, error) {
	if !isValidState(state) {
		return nil, fmt.Errorf("invalid application state: %s", state)
	}

	return &ApplicationState{value: state}, nil
}

// MustNewApplicationState creates a state, panicking on error
func MustNewApplicationState(state StateValue) *ApplicationState {
	appState, err := NewApplicationState(state)
	if err != nil {
		panic(fmt.Sprintf("cannot create state %s: %v", state, err))
	}
	return appState
}

// Value returns the state value
func (as *ApplicationState) Value() StateValue {
	return as.value
}

// String implements fmt.Stringer
func (as *ApplicationState) String() string {
	return string(as.value)
}

// Equal compares two states
func (as *ApplicationState) Equal(other *ApplicationState) bool {
	if other == nil {
		return false
	}
	return as.value == other.value
}

// IsRunning checks if the state is "running"
func (as *ApplicationState) IsRunning() bool {
	return as.value == StateRunning
}

// IsDeployed checks if the application is deployed (exists, running, or stopped)
func (as *ApplicationState) IsDeployed() bool {
	return as.value == StateExists ||
		as.value == StateRunning ||
		as.value == StateStopped
}

// IsError checks if the state is "error"
func (as *ApplicationState) IsError() bool {
	return as.value == StateError
}

// Description returns a human-readable description of the state
func (as *ApplicationState) Description() string {
	descriptions := map[StateValue]string{
		StateExists:  "Application exists in Dokku",
		StateRunning: "Application is running",
		StateStopped: "Application is stopped",
		StateError:   "Application is in error state",
	}

	if desc, exists := descriptions[as.value]; exists {
		return desc
	}
	return string(as.value)
}

// isValidState checks if a state value is valid
func isValidState(state StateValue) bool {
	validStates := []StateValue{
		StateExists, StateRunning, StateStopped, StateError,
	}

	return slices.Contains(validStates, state)
}

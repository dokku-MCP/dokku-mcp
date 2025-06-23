package process

import "fmt"

// ProcessScale represents the scale of a process as a value object.
type ProcessScale struct {
	value int
}

// NewProcessScale creates a new ProcessScale value object.
// It returns an error if the scale is negative.
func NewProcessScale(value int) (*ProcessScale, error) {
	if value < 0 {
		return nil, fmt.Errorf("process scale cannot be negative")
	}
	return &ProcessScale{value: value}, nil
}

// Value returns the int representation of the process scale.
func (s *ProcessScale) Value() int {
	return s.value
}

// Equal checks if two ProcessScale objects are equal.
func (s *ProcessScale) Equal(other *ProcessScale) bool {
	if other == nil {
		return false
	}
	return s.value == other.value
}

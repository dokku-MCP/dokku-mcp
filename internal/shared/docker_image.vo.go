package shared

import (
	"fmt"
	"regexp"
)

var (
	// DockerImageRegex defines the validation for a Docker image name.
	// This regex is a simplified version and might not cover all edge cases,
	// but it's a good starting point for validation.
	// It allows for optional host, user/org, repository and tag.
	DockerImageRegex = regexp.MustCompile(`^(?:[a-z0-9]+(?:[._-][a-z0-9]+)*\/)*[a-z0-9_.-]+(?::[a-zA-Z0-9_.-]+)?$`)
)

// DockerImage represents a Docker image name as a value object.
type DockerImage struct {
	value string
}

// NewDockerImage creates a new DockerImage value object.
// It returns an error if the image name is invalid.
func NewDockerImage(value string) (*DockerImage, error) {
	if !DockerImageRegex.MatchString(value) {
		return nil, fmt.Errorf("invalid docker image format: %s", value)
	}
	return &DockerImage{value: value}, nil
}

// Value returns the string representation of the Docker image.
func (d *DockerImage) Value() string {
	return d.value
}

// Equal checks if two DockerImage objects are equal.
func (d *DockerImage) Equal(other *DockerImage) bool {
	if other == nil {
		return false
	}
	return d.value == other.value
}

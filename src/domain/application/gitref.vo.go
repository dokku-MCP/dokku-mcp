package application

import (
	"fmt"
	"regexp"
	"strings"
)

type GitRef struct {
	value string
}

var (
	gitRefPattern = regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)
	shaPattern    = regexp.MustCompile(`^[a-f0-9]{7,40}$`)
)

func NewGitRef(ref string) (*GitRef, error) {
	if ref == "" {
		ref = "main"
	}

	ref = strings.TrimSpace(ref)

	if err := validateGitRef(ref); err != nil {
		return nil, fmt.Errorf("référence Git invalide: %w", err)
	}

	return &GitRef{value: ref}, nil
}

func MustNewGitRef(ref string) *GitRef {
	gitRef, err := NewGitRef(ref)
	if err != nil {
		panic(err)
	}
	return gitRef
}

func (g *GitRef) Value() string {
	return g.value
}

func (g *GitRef) IsSHA() bool {
	return shaPattern.MatchString(g.value)
}

func (g *GitRef) IsBranch() bool {
	return !g.IsSHA() && !g.IsTag()
}

func (g *GitRef) IsTag() bool {
	return strings.HasPrefix(g.value, "v") || strings.Contains(g.value, ".")
}

func (g *GitRef) String() string {
	return g.value
}

func (g *GitRef) Equal(other *GitRef) bool {
	if other == nil {
		return false
	}
	return g.value == other.value
}

func (g *GitRef) ShortSHA() string {
	if g.IsSHA() && len(g.value) >= 8 {
		return g.value[:8]
	}
	return g.value
}

func validateGitRef(ref string) error {
	if len(ref) == 0 {
		return fmt.Errorf("Reference cannot be empty")
	}

	if len(ref) > 250 {
		return fmt.Errorf("Reference is too long (max 250 characters)")
	}

	if strings.Contains(ref, "..") {
		return fmt.Errorf("Reference cannot contain '..'")
	}

	if strings.HasPrefix(ref, "-") || strings.HasSuffix(ref, "-") {
		return fmt.Errorf("Reference cannot start or end with '-'")
	}

	if !gitRefPattern.MatchString(ref) {
		return fmt.Errorf("Reference contains invalid characters")
	}

	return nil
}

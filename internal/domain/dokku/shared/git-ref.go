package shared

import (
	"fmt"
	"regexp"
	"strings"
)

// GitRef représente une référence Git valide (branch, tag, commit)
type GitRef struct {
	value   string
	refType GitRefType
}

// GitRefType représente le type de référence Git
type GitRefType string

const (
	GitRefTypeBranch GitRefType = "branch"
	GitRefTypeTag    GitRefType = "tag"
	GitRefTypeCommit GitRefType = "commit"
)

var (
	// Pattern pour détecter un hash de commit (au moins 7 caractères hexadécimaux)
	commitHashPattern = regexp.MustCompile(`^[a-fA-F0-9]{7,40}$`)
	// Pattern pour détecter un tag (commençant souvent par v)
	tagPattern = regexp.MustCompile(`^v?\d+\.\d+(\.\d+)?(-[a-zA-Z0-9.-]+)?$`)
)

// NewGitRef crée une nouvelle référence Git avec validation
func NewGitRef(ref string) (*GitRef, error) {
	ref = strings.TrimSpace(ref)

	if err := validateGitRef(ref); err != nil {
		return nil, fmt.Errorf("référence Git invalide: %w", err)
	}

	refType := detectGitRefType(ref)

	return &GitRef{
		value:   ref,
		refType: refType,
	}, nil
}

// MustNewGitRef crée une référence Git en paniquant en cas d'erreur
func MustNewGitRef(ref string) *GitRef {
	gitRef, err := NewGitRef(ref)
	if err != nil {
		panic(fmt.Sprintf("impossible de créer la référence Git %s: %v", ref, err))
	}
	return gitRef
}

// NewBranchRef crée une référence de branche
func NewBranchRef(branch string) (*GitRef, error) {
	gitRef, err := NewGitRef(branch)
	if err != nil {
		return nil, err
	}
	gitRef.refType = GitRefTypeBranch
	return gitRef, nil
}

// NewTagRef crée une référence de tag
func NewTagRef(tag string) (*GitRef, error) {
	gitRef, err := NewGitRef(tag)
	if err != nil {
		return nil, err
	}
	gitRef.refType = GitRefTypeTag
	return gitRef, nil
}

// NewCommitRef crée une référence de commit
func NewCommitRef(commit string) (*GitRef, error) {
	if !commitHashPattern.MatchString(commit) {
		return nil, fmt.Errorf("hash de commit invalide: %s", commit)
	}

	gitRef, err := NewGitRef(commit)
	if err != nil {
		return nil, err
	}
	gitRef.refType = GitRefTypeCommit
	return gitRef, nil
}

// Value retourne la valeur de la référence
func (g *GitRef) Value() string {
	return g.value
}

// Type retourne le type de référence
func (g *GitRef) Type() GitRefType {
	return g.refType
}

// IsBranch vérifie si c'est une branche
func (g *GitRef) IsBranch() bool {
	return g.refType == GitRefTypeBranch
}

// IsTag vérifie si c'est un tag
func (g *GitRef) IsTag() bool {
	return g.refType == GitRefTypeTag
}

// IsCommit vérifie si c'est un commit
func (g *GitRef) IsCommit() bool {
	return g.refType == GitRefTypeCommit
}

// IsMainBranch vérifie si c'est une branche principale
func (g *GitRef) IsMainBranch() bool {
	if !g.IsBranch() {
		return false
	}

	mainBranches := []string{"main", "master", "develop", "dev"}
	for _, branch := range mainBranches {
		if g.value == branch {
			return true
		}
	}
	return false
}

// String implémente fmt.Stringer
func (g *GitRef) String() string {
	return g.value
}

// Equal compare deux références Git
func (g *GitRef) Equal(other *GitRef) bool {
	if other == nil {
		return false
	}
	return g.value == other.value && g.refType == other.refType
}

// validateGitRef valide une référence Git
func validateGitRef(ref string) error {
	if ref == "" {
		return fmt.Errorf("la référence Git ne peut pas être vide")
	}

	if len(ref) > 250 {
		return fmt.Errorf("la référence Git est trop longue (max 250 caractères)")
	}

	// Caractères interdits dans Git
	forbiddenChars := []string{" ", "\t", "\n", "\r", "~", "^", ":", "?", "*", "[", "\\"}
	for _, char := range forbiddenChars {
		if strings.Contains(ref, char) {
			return fmt.Errorf("la référence Git contient des caractères interdits: %s", char)
		}
	}

	// Ne peut pas commencer ou finir par un point
	if strings.HasPrefix(ref, ".") || strings.HasSuffix(ref, ".") {
		return fmt.Errorf("la référence Git ne peut pas commencer ou finir par un point")
	}

	// Ne peut pas commencer ou finir par un slash
	if strings.HasPrefix(ref, "/") || strings.HasSuffix(ref, "/") {
		return fmt.Errorf("la référence Git ne peut pas commencer ou finir par un slash")
	}

	// Ne peut pas contenir des doubles slashes
	if strings.Contains(ref, "//") {
		return fmt.Errorf("la référence Git ne peut pas contenir des doubles slashes")
	}

	return nil
}

// detectGitRefType détecte automatiquement le type de référence
func detectGitRefType(ref string) GitRefType {
	// Si c'est un hash de commit
	if commitHashPattern.MatchString(ref) {
		return GitRefTypeCommit
	}

	// Si ça ressemble à un tag
	if tagPattern.MatchString(ref) {
		return GitRefTypeTag
	}

	// Par défaut, c'est une branche
	return GitRefTypeBranch
}

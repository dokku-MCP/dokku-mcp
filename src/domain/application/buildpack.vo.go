package application

import (
	"fmt"
	"regexp"
	"strings"
)

// BuildpackName représente un nom de buildpack valide
type BuildpackName struct {
	value string
}

var (
	// Buildpacks officiels supportés
	OfficialBuildpacks = map[string]string{
		"node":       "heroku/nodejs",
		"python":     "heroku/python",
		"ruby":       "heroku/ruby",
		"java":       "heroku/java",
		"php":        "heroku/php",
		"go":         "heroku/go",
		"static":     "dokku/static",
		"dockerfile": "dockerfile",
		"multi":      "multi",
	}

	// Pattern pour valider les URLs de buildpack
	buildpackURLPattern  = regexp.MustCompile(`^https?://[a-zA-Z0-9.-]+/[a-zA-Z0-9._/-]+$`)
	buildpackNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)
)

// NewBuildpackName crée un nouveau nom de buildpack avec validation
func NewBuildpackName(name string) (*BuildpackName, error) {
	name = strings.TrimSpace(name)

	if err := validateBuildpackName(name); err != nil {
		return nil, fmt.Errorf("nom de buildpack invalide: %w", err)
	}

	return &BuildpackName{value: name}, nil
}

// MustNewBuildpackName crée un buildpack en paniquant en cas d'erreur
func MustNewBuildpackName(name string) *BuildpackName {
	bp, err := NewBuildpackName(name)
	if err != nil {
		panic(err)
	}
	return bp
}

// Value retourne la valeur du buildpack
func (b *BuildpackName) Value() string {
	return b.value
}

// IsOfficial vérifie si c'est un buildpack officiel
func (b *BuildpackName) IsOfficial() bool {
	// Vérifier par nom court
	if _, exists := OfficialBuildpacks[b.value]; exists {
		return true
	}

	// Vérifier par nom complet
	for _, officialName := range OfficialBuildpacks {
		if b.value == officialName {
			return true
		}
	}

	return false
}

// IsURL vérifie si c'est une URL de buildpack
func (b *BuildpackName) IsURL() bool {
	return buildpackURLPattern.MatchString(b.value)
}

// IsDockerfile vérifie si c'est le buildpack Dockerfile
func (b *BuildpackName) IsDockerfile() bool {
	return strings.ToLower(b.value) == "dockerfile"
}

// IsMulti vérifie si c'est le buildpack multi
func (b *BuildpackName) IsMulti() bool {
	return strings.ToLower(b.value) == "multi"
}

// ExpandName retourne le nom complet du buildpack
func (b *BuildpackName) ExpandName() string {
	if expanded, exists := OfficialBuildpacks[b.value]; exists {
		return expanded
	}
	return b.value
}

// GetLanguage retourne le langage détecté du buildpack
func (b *BuildpackName) GetLanguage() string {
	// Pour les buildpacks officiels
	for lang, fullName := range OfficialBuildpacks {
		if b.value == lang || b.value == fullName {
			return lang
		}
	}

	// Pour les buildpacks custom, essayer de détecter depuis le nom
	lower := strings.ToLower(b.value)
	if strings.Contains(lower, "node") || strings.Contains(lower, "javascript") {
		return "node"
	}
	if strings.Contains(lower, "python") {
		return "python"
	}
	if strings.Contains(lower, "ruby") {
		return "ruby"
	}
	if strings.Contains(lower, "java") {
		return "java"
	}
	if strings.Contains(lower, "php") {
		return "php"
	}
	if strings.Contains(lower, "go") || strings.Contains(lower, "golang") {
		return "go"
	}

	return "unknown"
}

// String implémente fmt.Stringer
func (b *BuildpackName) String() string {
	return b.value
}

// Equal compare deux noms de buildpack
func (b *BuildpackName) Equal(other *BuildpackName) bool {
	if other == nil {
		return false
	}
	return b.value == other.value
}

// validateBuildpackName valide un nom de buildpack
func validateBuildpackName(name string) error {
	if name == "" {
		return fmt.Errorf("le nom de buildpack ne peut pas être vide")
	}

	if len(name) > 200 {
		return fmt.Errorf("le nom de buildpack est trop long (max 200 caractères)")
	}

	// Cas spéciaux valides
	if name == "dockerfile" || name == "multi" {
		return nil
	}

	// URL valide
	if buildpackURLPattern.MatchString(name) {
		return nil
	}

	// Nom de buildpack valide
	if !buildpackNamePattern.MatchString(name) {
		return fmt.Errorf("le nom de buildpack contient des caractères invalides")
	}

	// Vérifications de sécurité
	if strings.Contains(name, "..") {
		return fmt.Errorf("le nom de buildpack ne peut pas contenir '..'")
	}

	return nil
}

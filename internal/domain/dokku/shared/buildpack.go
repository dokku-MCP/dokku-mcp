package shared

import (
	"fmt"
	"regexp"
	"strings"
)

// BuildpackName représente un nom de buildpack valide pour Dokku
type BuildpackName struct {
	value string
}

// BuildpackInfo contient les informations détaillées d'un buildpack
type BuildpackInfo struct {
	name        *BuildpackName
	language    string
	isOfficial  bool
	isURL       bool
	description string
}

var (
	// Buildpacks officiels supportés par Dokku
	OfficialBuildpacks = map[string]BuildpackSpec{
		"node":       {FullName: "heroku/nodejs", Language: "javascript", Description: "Node.js et npm"},
		"python":     {FullName: "heroku/python", Language: "python", Description: "Python avec pip"},
		"ruby":       {FullName: "heroku/ruby", Language: "ruby", Description: "Ruby avec bundler"},
		"java":       {FullName: "heroku/java", Language: "java", Description: "Java avec Maven/Gradle"},
		"php":        {FullName: "heroku/php", Language: "php", Description: "PHP avec Composer"},
		"go":         {FullName: "heroku/go", Language: "go", Description: "Go avec plugins"},
		"static":     {FullName: "dokku/static", Language: "static", Description: "Sites statiques"},
		"dockerfile": {FullName: "dockerfile", Language: "docker", Description: "Dockerfile personnalisé"},
		"multi":      {FullName: "multi", Language: "multi", Description: "Multi-buildpack"},
	}

	// Patterns de validation
	buildpackURLPattern  = regexp.MustCompile(`^https?://[a-zA-Z0-9.-]+/[a-zA-Z0-9._/-]+$`)
	buildpackNamePattern = regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)
)

// BuildpackSpec spécification d'un buildpack officiel
type BuildpackSpec struct {
	FullName    string
	Language    string
	Description string
}

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
		panic(fmt.Sprintf("impossible de créer le buildpack %s: %v", name, err))
	}
	return bp
}

// NewBuildpackInfo crée des informations complètes de buildpack
func NewBuildpackInfo(name string) (*BuildpackInfo, error) {
	buildpackName, err := NewBuildpackName(name)
	if err != nil {
		return nil, err
	}

	language := detectLanguage(name)
	isOfficial := isOfficialBuildpack(name)
	isURL := buildpackURLPattern.MatchString(name)
	description := generateDescription(name, language, isOfficial)

	return &BuildpackInfo{
		name:        buildpackName,
		language:    language,
		isOfficial:  isOfficial,
		isURL:       isURL,
		description: description,
	}, nil
}

// Value retourne la valeur du buildpack
func (b *BuildpackName) Value() string {
	return b.value
}

// IsOfficial vérifie si c'est un buildpack officiel
func (b *BuildpackName) IsOfficial() bool {
	return isOfficialBuildpack(b.value)
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
	if spec, exists := OfficialBuildpacks[b.value]; exists {
		return spec.FullName
	}
	return b.value
}

// GetLanguage retourne le langage détecté du buildpack
func (b *BuildpackName) GetLanguage() string {
	return detectLanguage(b.value)
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

// Méthodes pour BuildpackInfo

// Name retourne le nom du buildpack
func (bi *BuildpackInfo) Name() *BuildpackName {
	return bi.name
}

// Language retourne le langage
func (bi *BuildpackInfo) Language() string {
	return bi.language
}

// IsOfficial retourne si c'est officiel
func (bi *BuildpackInfo) IsOfficial() bool {
	return bi.isOfficial
}

// IsURL retourne si c'est une URL
func (bi *BuildpackInfo) IsURL() bool {
	return bi.isURL
}

// Description retourne la description
func (bi *BuildpackInfo) Description() string {
	return bi.description
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

// isOfficialBuildpack vérifie si un buildpack est officiel
func isOfficialBuildpack(name string) bool {
	// Vérifier par nom court
	if _, exists := OfficialBuildpacks[name]; exists {
		return true
	}

	// Vérifier par nom complet
	for _, spec := range OfficialBuildpacks {
		if name == spec.FullName {
			return true
		}
	}

	return false
}

// detectLanguage détecte le langage d'un buildpack
func detectLanguage(name string) string {
	// Pour les buildpacks officiels
	for shortName, spec := range OfficialBuildpacks {
		if name == shortName || name == spec.FullName {
			return spec.Language
		}
	}

	// Pour les buildpacks custom, essayer de détecter depuis le nom
	lower := strings.ToLower(name)

	languagePatterns := map[string][]string{
		"javascript": {"node", "javascript", "js"},
		"python":     {"python", "py"},
		"ruby":       {"ruby", "rb"},
		"java":       {"java", "maven", "gradle"},
		"php":        {"php", "composer"},
		"go":         {"go", "golang"},
		"rust":       {"rust", "cargo"},
		"dotnet":     {"dotnet", "csharp", "fsharp"},
		"scala":      {"scala", "sbt"},
		"elixir":     {"elixir", "phoenix"},
		"clojure":    {"clojure", "lein"},
	}

	for language, patterns := range languagePatterns {
		for _, pattern := range patterns {
			if strings.Contains(lower, pattern) {
				return language
			}
		}
	}

	return "unknown"
}

// generateDescription génère une description pour le buildpack
func generateDescription(name, language string, isOfficial bool) string {
	if spec, exists := OfficialBuildpacks[name]; exists {
		return spec.Description
	}

	if isOfficial {
		return fmt.Sprintf("Buildpack officiel pour %s", language)
	}

	if buildpackURLPattern.MatchString(name) {
		return fmt.Sprintf("Buildpack personnalisé (%s)", language)
	}

	return fmt.Sprintf("Buildpack %s", language)
}

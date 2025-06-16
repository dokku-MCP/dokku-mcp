package application

import (
	"fmt"
	"regexp"
	"strings"
)

// ApplicationName représente un nom d'application Dokku valide
type ApplicationName struct {
	value string
}

var (
	// Pattern pour valider un nom d'application Dokku
	// Doit respecter les conventions DNS et Dokku
	applicationNamePattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)
)

// NewApplicationName crée un nouveau nom d'application avec validation
func NewApplicationName(name string) (*ApplicationName, error) {
	name = strings.TrimSpace(strings.ToLower(name))

	if err := validateApplicationName(name); err != nil {
		return nil, fmt.Errorf("invalid application name: %w", err)
	}

	return &ApplicationName{value: name}, nil
}

// MustNewApplicationName crée un nom d'application en paniquant en cas d'erreur
func MustNewApplicationName(name string) *ApplicationName {
	appName, err := NewApplicationName(name)
	if err != nil {
		panic(fmt.Sprintf("impossible de créer le nom d'application %s: %v", name, err))
	}
	return appName
}

// Value retourne la valeur du nom d'application
func (an *ApplicationName) Value() string {
	return an.value
}

// String implémente fmt.Stringer
func (an *ApplicationName) String() string {
	return an.value
}

// Equal compare deux noms d'application
func (an *ApplicationName) Equal(other *ApplicationName) bool {
	if other == nil {
		return false
	}
	return an.value == other.value
}

// IsReserved vérifie si le nom est un nom réservé par Dokku
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

// validateApplicationName valide un nom d'application selon les règles Dokku
func validateApplicationName(name string) error {
	if name == "" {
		return fmt.Errorf("le nom d'application ne peut pas être vide")
	}

	if len(name) > 63 {
		return fmt.Errorf("le nom d'application ne peut pas dépasser 63 caractères")
	}

	if len(name) < 1 {
		return fmt.Errorf("le nom d'application doit contenir au moins 1 caractère")
	}

	// Vérifier le pattern DNS
	if !applicationNamePattern.MatchString(name) {
		return fmt.Errorf("le nom d'application doit respecter le format DNS (lettres minuscules, chiffres, tirets)")
	}

	// Ne peut pas commencer ou finir par un tiret
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "-") {
		return fmt.Errorf("le nom d'application ne peut pas commencer ou finir par un tiret")
	}

	// Vérifier les noms réservés
	reservedNames := []string{
		"dokku", "tls", "app", "plugin", "plugins", "config", "logs",
		"ps", "run", "shell", "enter", "backup", "restore", "certs",
		"domains", "git", "storage", "network", "proxy", "apps",
		"service", "services", "builder", "scheduler", "registry",
	}

	for _, reserved := range reservedNames {
		if name == reserved {
			return fmt.Errorf("le nom '%s' est réservé par Dokku", name)
		}
	}

	return nil
}

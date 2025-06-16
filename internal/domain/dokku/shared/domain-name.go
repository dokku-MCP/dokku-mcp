package shared

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// DomainName représente un nom de domaine valide pour Dokku
type DomainName struct {
	value string
}

var (
	domainPattern    = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
	localhostPattern = regexp.MustCompile(`^localhost(:[0-9]+)?$`)
	ipPattern        = regexp.MustCompile(`^([0-9]{1,3}\.){3}[0-9]{1,3}(:[0-9]+)?$`)
)

// NewDomainName crée un nouveau nom de domaine avec validation
func NewDomainName(domain string) (*DomainName, error) {
	domain = strings.TrimSpace(strings.ToLower(domain))

	if err := validateDomain(domain); err != nil {
		return nil, fmt.Errorf("invalid domain name: %w", err)
	}

	return &DomainName{value: domain}, nil
}

// MustNewDomainName crée un domaine en paniquant en cas d'erreur
func MustNewDomainName(domain string) *DomainName {
	d, err := NewDomainName(domain)
	if err != nil {
		panic(fmt.Sprintf("impossible de créer le domaine %s: %v", domain, err))
	}
	return d
}

// Value retourne la valeur du domaine
func (d *DomainName) Value() string {
	return d.value
}

// IsLocalhost vérifie si c'est localhost
func (d *DomainName) IsLocalhost() bool {
	return localhostPattern.MatchString(d.value)
}

// IsIP vérifie si c'est une adresse IP
func (d *DomainName) IsIP() bool {
	return ipPattern.MatchString(d.value)
}

// IsWildcard vérifie si c'est un domaine wildcard
func (d *DomainName) IsWildcard() bool {
	return strings.HasPrefix(d.value, "*.")
}

// IsSubdomain vérifie si c'est un sous-domaine
func (d *DomainName) IsSubdomain() bool {
	parts := strings.Split(d.value, ".")
	return len(parts) > 2 && !d.IsWildcard()
}

// RootDomain retourne le domaine racine
func (d *DomainName) RootDomain() string {
	if d.IsIP() || d.IsLocalhost() {
		return d.value
	}

	parts := strings.Split(d.value, ".")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], ".")
	}
	return d.value
}

// WithPort ajoute un port au domaine
func (d *DomainName) WithPort(port int) (*DomainName, error) {
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("port invalide: %d", port)
	}

	domain := strings.Split(d.value, ":")[0]
	return NewDomainName(fmt.Sprintf("%s:%d", domain, port))
}

// ToURL convertit le domaine en URL
func (d *DomainName) ToURL(scheme string) (*url.URL, error) {
	if scheme == "" {
		scheme = "https"
	}

	urlStr := fmt.Sprintf("%s://%s", scheme, d.value)
	return url.Parse(urlStr)
}

// String implémente fmt.Stringer
func (d *DomainName) String() string {
	return d.value
}

// Equal compare deux noms de domaine
func (d *DomainName) Equal(other *DomainName) bool {
	if other == nil {
		return false
	}
	return d.value == other.value
}

// validateDomain valide un nom de domaine
func validateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("le domaine ne peut pas être vide")
	}

	if len(domain) > 253 {
		return fmt.Errorf("le domaine est trop long (max 253 caractères)")
	}

	// Localhost et IP sont acceptés
	if localhostPattern.MatchString(domain) || ipPattern.MatchString(domain) {
		return nil
	}

	if !domainPattern.MatchString(domain) {
		return fmt.Errorf("format de domaine invalide")
	}

	parts := strings.Split(domain, ".")
	for _, part := range parts {
		if len(part) > 63 {
			return fmt.Errorf("label trop long dans le domaine: %s", part)
		}
		if part == "" {
			return fmt.Errorf("label vide dans le domaine")
		}
	}

	// Vérifier le TLD si c'est un domaine complet
	if len(parts) > 1 {
		tld := parts[len(parts)-1]
		if len(tld) < 2 {
			return fmt.Errorf("TLD trop court: %s", tld)
		}
	}

	return nil
}

package application

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

type DomainName struct {
	value string
}

var (
	domainPattern    = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
	localhostPattern = regexp.MustCompile(`^localhost(:[0-9]+)?$`)
	ipPattern        = regexp.MustCompile(`^([0-9]{1,3}\.){3}[0-9]{1,3}(:[0-9]+)?$`)
)

func NewDomain(domain string) (*DomainName, error) {
	domain = strings.TrimSpace(strings.ToLower(domain))

	if err := validateDomain(domain); err != nil {
		return nil, fmt.Errorf("domaine invalide: %w", err)
	}

	return &DomainName{value: domain}, nil
}

func MustNewDomain(domain string) *DomainName {
	d, err := NewDomain(domain)
	if err != nil {
		panic(err)
	}
	return d
}

func (d *DomainName) Value() string {
	return d.value
}

func (d *DomainName) IsLocalhost() bool {
	return localhostPattern.MatchString(d.value)
}

func (d *DomainName) IsIP() bool {
	return ipPattern.MatchString(d.value)
}

func (d *DomainName) IsWildcard() bool {
	return strings.HasPrefix(d.value, "*.")
}

func (d *DomainName) IsSubdomain() bool {
	parts := strings.Split(d.value, ".")
	return len(parts) > 2 && !d.IsWildcard()
}

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

func (d *DomainName) WithPort(port int) (*DomainName, error) {
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid port: %d", port)
	}

	domain := strings.Split(d.value, ":")[0]
	return NewDomain(fmt.Sprintf("%s:%d", domain, port))
}

func (d *DomainName) ToURL(scheme string) (*url.URL, error) {
	if scheme == "" {
		scheme = "https"
	}

	urlStr := fmt.Sprintf("%s://%s", scheme, d.value)
	return url.Parse(urlStr)
}

func (d *DomainName) String() string {
	return d.value
}

func (d *DomainName) Equal(other *DomainName) bool {
	if other == nil {
		return false
	}
	return d.value == other.value
}

func validateDomain(domain string) error {
	if domain == "" {
		return fmt.Errorf("Domain cannot be empty")
	}

	if len(domain) > 253 {
		return fmt.Errorf("Domain is too long (max 253 characters)")
	}

	if localhostPattern.MatchString(domain) || ipPattern.MatchString(domain) {
		return nil // Localhost and IP are accepted
	}

	if !domainPattern.MatchString(domain) {
		return fmt.Errorf("invalid domain format")
	}

	parts := strings.Split(domain, ".")
	for _, part := range parts {
		if len(part) > 63 {
			return fmt.Errorf("label too long in domain: %s", part)
		}
		if part == "" {
			return fmt.Errorf("empty label in domain")
		}
	}

	if len(parts) > 1 {
		tld := parts[len(parts)-1]
		if len(tld) < 2 {
			return fmt.Errorf("TLD too short: %s", tld)
		}
	}

	return nil
}

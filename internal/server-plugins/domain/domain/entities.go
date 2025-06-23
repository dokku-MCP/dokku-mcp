package domain

import (
	"time"
)

// GlobalDomain represents global domain configuration
type GlobalDomain struct {
	Domain     string    `json:"domain"`
	IsWildcard bool      `json:"is_wildcard"`
	AddedAt    time.Time `json:"added_at"`
}

// DomainsReport represents the structured output of a domains report
type DomainsReport struct {
	Global      DomainsReportSection            `json:"global"`
	PerApp      map[string]DomainsReportSection `json:"per_app"`
	RawOutput   string                          `json:"raw_output"`
	GeneratedAt time.Time                       `json:"generated_at"`
}

// DomainsReportSection represents a section of the domains report
type DomainsReportSection struct {
	Domains      []string `json:"domains"`
	Vhosts       []string `json:"vhosts"`
	ProxyEnabled bool     `json:"proxy_enabled"`
	ProxyType    string   `json:"proxy_type,omitempty"`
	AppName      string   `json:"app_name,omitempty"`
	SSLEnabled   bool     `json:"ssl_enabled"`
	SSLCertPath  string   `json:"ssl_cert_path,omitempty"`
	SSLKeyPath   string   `json:"ssl_key_path,omitempty"`
	SSLHostname  string   `json:"ssl_hostname,omitempty"`
	LetsEncrypt  bool     `json:"letsencrypt"`
}

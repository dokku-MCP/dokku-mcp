package domain

import (
	"context"
)

// DomainRepository defines methods for managing global domains
type DomainRepository interface {
	ListGlobalDomains(ctx context.Context) ([]GlobalDomain, error)
	AddGlobalDomain(ctx context.Context, domain string) error
	RemoveGlobalDomain(ctx context.Context, domain string) error
	SetGlobalDomains(ctx context.Context, domains []string) error
	ClearGlobalDomains(ctx context.Context) error
	GetDomainsReport(ctx context.Context) (*DomainsReport, error)
}

package application

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/domain/domain"
)

// DomainService provides application-level orchestration for domain functionality
type DomainService struct {
	domainRepo domain.DomainRepository
	logger     *slog.Logger
}

// NewDomainService creates a new domain application service
func NewDomainService(
	domainRepo domain.DomainRepository,
	logger *slog.Logger,
) *DomainService {
	return &DomainService{
		domainRepo: domainRepo,
		logger:     logger,
	}
}

// ListGlobalDomains lists all global domains
func (s *DomainService) ListGlobalDomains(ctx context.Context) ([]domain.GlobalDomain, error) {
	s.logger.Debug("Listing global domains")
	return s.domainRepo.ListGlobalDomains(ctx)
}

// AddGlobalDomain adds a new global domain
func (s *DomainService) AddGlobalDomain(ctx context.Context, domainName string) error {
	s.logger.Info("Adding global domain", "domain", domainName)
	if err := s.validateDomainName(domainName); err != nil {
		return fmt.Errorf("invalid domain name: %w", err)
	}
	return s.domainRepo.AddGlobalDomain(ctx, domainName)
}

// RemoveGlobalDomain removes a global domain
func (s *DomainService) RemoveGlobalDomain(ctx context.Context, domainName string) error {
	s.logger.Info("Removing global domain", "domain", domainName)
	if domainName == "" {
		return fmt.Errorf("domain name cannot be empty")
	}
	return s.domainRepo.RemoveGlobalDomain(ctx, domainName)
}

// SetGlobalDomains sets all global domains, replacing existing ones
func (s *DomainService) SetGlobalDomains(ctx context.Context, domains []string) error {
	s.logger.Info("Setting global domains", "domains", domains)
	for _, domainName := range domains {
		if err := s.validateDomainName(domainName); err != nil {
			return fmt.Errorf("invalid domain name '%s': %w", domainName, err)
		}
	}
	return s.domainRepo.SetGlobalDomains(ctx, domains)
}

// ClearGlobalDomains clears all global domains
func (s *DomainService) ClearGlobalDomains(ctx context.Context) error {
	s.logger.Info("Clearing all global domains")
	return s.domainRepo.ClearGlobalDomains(ctx)
}

// GetDomainsReport gets the full domains report
func (s *DomainService) GetDomainsReport(ctx context.Context) (*domain.DomainsReport, error) {
	s.logger.Debug("Getting domains report")
	return s.domainRepo.GetDomainsReport(ctx)
}

func (s *DomainService) validateDomainName(domain string) error {
	if domain == "" {
		return fmt.Errorf("domain name cannot be empty")
	}
	if len(domain) > 253 {
		return fmt.Errorf("domain name too long (max 253 characters)")
	}
	return nil
}

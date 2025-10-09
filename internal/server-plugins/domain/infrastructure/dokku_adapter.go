package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	dokkuApi "github.com/alex-galey/dokku-mcp/internal/dokku-api"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/domain/domain"
)

// DokkuDomainAdapter implements the domain repository using Dokku CLI
type DokkuDomainAdapter struct {
	client dokkuApi.DokkuClient
	logger *slog.Logger
}

// NewDokkuDomainAdapter creates a new domain adapter
func NewDokkuDomainAdapter(client dokkuApi.DokkuClient, logger *slog.Logger) domain.DomainRepository {
	return &DokkuDomainAdapter{
		client: client,
		logger: logger,
	}
}

// executeCommand wraps the client's ExecuteCommand with domain-specific context and validation
func (a *DokkuDomainAdapter) executeCommand(ctx context.Context, command domain.DomainCommand, args []string) ([]byte, error) {
	if !command.IsValid() {
		return nil, fmt.Errorf("invalid domain command: %s", command)
	}
	return a.client.ExecuteCommand(ctx, command.String(), args)
}

func (a *DokkuDomainAdapter) SetLetsEncryptEmail(ctx context.Context, email string) error {
	_, err := a.executeCommand(ctx, domain.CommandLetsEncryptSet, []string{"--global", "email", email})
	if err != nil {
		return fmt.Errorf("failed to set letsencrypt email: %w", err)
	}
	return nil
}

// ListGlobalDomains retrieves global domains
func (a *DokkuDomainAdapter) ListGlobalDomains(ctx context.Context) ([]domain.GlobalDomain, error) {
	output, err := a.executeCommand(ctx, domain.CommandDomainsReport, []string{"--global"})
	if err != nil {
		return nil, fmt.Errorf("failed to list global domains: %w", err)
	}

	domainStrings := a.parseGlobalDomains(string(output))
	domains := make([]domain.GlobalDomain, 0, len(domainStrings))

	for _, domainStr := range domainStrings {
		domains = append(domains, domain.GlobalDomain{
			Domain:     domainStr,
			IsWildcard: strings.HasPrefix(domainStr, "*."),
			AddedAt:    time.Now(), // We can't get the actual added time from Dokku
		})
	}

	return domains, nil
}

// AddGlobalDomain adds a global domain
func (a *DokkuDomainAdapter) AddGlobalDomain(ctx context.Context, domainName string) error {
	_, err := a.executeCommand(ctx, domain.CommandDomainsAddGlobal, []string{domainName})
	if err != nil {
		return fmt.Errorf("failed to add global domain %s: %w", domainName, err)
	}
	return nil
}

// RemoveGlobalDomain removes a global domain
func (a *DokkuDomainAdapter) RemoveGlobalDomain(ctx context.Context, domainName string) error {
	_, err := a.executeCommand(ctx, domain.CommandDomainsRemoveGlobal, []string{domainName})
	if err != nil {
		return fmt.Errorf("failed to remove global domain %s: %w", domainName, err)
	}
	return nil
}

// SetGlobalDomains sets global domains
func (a *DokkuDomainAdapter) SetGlobalDomains(ctx context.Context, domains []string) error {
	_, err := a.executeCommand(ctx, domain.CommandDomainsSetGlobal, domains)
	if err != nil {
		return fmt.Errorf("failed to set global domains: %w", err)
	}
	return nil
}

// ClearGlobalDomains clears all global domains
func (a *DokkuDomainAdapter) ClearGlobalDomains(ctx context.Context) error {
	_, err := a.executeCommand(ctx, domain.CommandDomainsClearGlobal, []string{})
	if err != nil {
		return fmt.Errorf("failed to clear global domains: %w", err)
	}
	return nil
}

// GetDomainsReport retrieves the full domains report
func (a *DokkuDomainAdapter) GetDomainsReport(ctx context.Context) (*domain.DomainsReport, error) {
	output, err := a.executeCommand(ctx, domain.CommandDomainsReport, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to get domains report: %w", err)
	}

	report := &domain.DomainsReport{
		RawOutput:   string(output),
		GeneratedAt: time.Now(),
		PerApp:      make(map[string]domain.DomainsReportSection),
	}

	lines := dokkuApi.ParseTrimmedLines(string(output), false)
	var currentApp string
	var currentSection *domain.DomainsReportSection

	for _, line := range lines {
		if strings.HasPrefix(line, "=====>") {
			if currentApp != "" && currentSection != nil {
				appSection := *currentSection
				report.PerApp[currentApp] = appSection
			}
			parts := strings.Fields(line)
			if len(parts) > 1 {
				currentApp = parts[1]
				currentSection = &domain.DomainsReportSection{AppName: currentApp}
			}
		} else if currentSection != nil {
			a.parseReportLine(line, currentSection)
		} else {
			a.parseReportLine(line, &report.Global)
		}
	}
	if currentApp != "" && currentSection != nil {
		appSection := *currentSection
		report.PerApp[currentApp] = appSection
	}

	return report, nil
}

func (a *DokkuDomainAdapter) parseGlobalDomains(output string) []string {
	lines := dokkuApi.ParseTrimmedLines(output, true)

	for _, line := range lines {
		if strings.Contains(line, "Global vhosts:") {
			idx := strings.Index(line, ":")
			if idx != -1 {
				domainsStr := strings.TrimSpace(line[idx+1:])
				if domainsStr != "" && domainsStr != "none" {
					return strings.Fields(domainsStr)
				}
			}
			break
		}
	}
	return []string{}
}

func (a *DokkuDomainAdapter) parseReportLine(line string, section *domain.DomainsReportSection) {
	key, value, ok := dokkuApi.ParseColonKeyValueLine(line)
	if !ok {
		return
	}

	switch key {
	case "Domains app vhosts":
		if value != "none" {
			section.Vhosts = strings.Fields(value)
		}
	case "Domains global vhosts":
		if value != "none" {
			section.Vhosts = append(section.Vhosts, strings.Fields(value)...)
		}
	case "Domains enabled":
		section.ProxyEnabled = (value == "true")
	case "Domains proxy type":
		section.ProxyType = value
	}
}

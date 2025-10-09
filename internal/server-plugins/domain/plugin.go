package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	dokkuApi "github.com/alex-galey/dokku-mcp/internal/dokku-api"
	"github.com/alex-galey/dokku-mcp/internal/server"
	serverDomain "github.com/alex-galey/dokku-mcp/internal/server-plugin/domain"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/domain/application"
	"github.com/alex-galey/dokku-mcp/internal/server-plugins/domain/infrastructure"
	"github.com/mark3labs/mcp-go/mcp"
)

// DomainServerPlugin provides domain management functionality
type DomainServerPlugin struct {
	domainService *application.DomainService
	logger        *slog.Logger
}

// NewDomainServerPlugin creates a new domain server plugin
func NewDomainServerPlugin(client dokkuApi.DokkuClient, logger *slog.Logger) serverDomain.ServerPlugin {
	adapter := infrastructure.NewDokkuDomainAdapter(client, logger)
	domainService := application.NewDomainService(adapter, logger)
	return &DomainServerPlugin{
		domainService: domainService,
		logger:        logger,
	}
}

func (p *DomainServerPlugin) ID() string   { return "domain" }
func (p *DomainServerPlugin) Name() string { return "Dokku Domains" }
func (p *DomainServerPlugin) Description() string {
	return "Manages global and application-specific domains"
}
func (p *DomainServerPlugin) Version() string         { return "0.1.0" }
func (p *DomainServerPlugin) DokkuPluginName() string { return "domains" }

// ResourceProvider implementation
func (p *DomainServerPlugin) GetResources(ctx context.Context) ([]serverDomain.Resource, error) {
	return []serverDomain.Resource{
		{
			URI:         "dokku://domains/report",
			Name:        "Domains Report",
			Description: "Report of all domains configured in Dokku",
			MIMEType:    "application/json",
			Handler:     p.handleDomainsReportResource,
		},
	}, nil
}

// ToolProvider implementation
func (p *DomainServerPlugin) GetTools(ctx context.Context) ([]serverDomain.Tool, error) {
	return []serverDomain.Tool{
		{
			Name:        "list_global_domains",
			Description: "List all global domains",
			Builder:     p.buildListGlobalDomainsTool,
			Handler:     p.handleListGlobalDomains,
		},
		{
			Name:        "add_global_domain",
			Description: "Add a global domain",
			Builder:     p.buildAddGlobalDomainTool,
			Handler:     p.handleAddGlobalDomain,
		},
	}, nil
}

func (p *DomainServerPlugin) handleDomainsReportResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	report, err := p.domainService.GetDomainsReport(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get domains report: %w", err)
	}
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to serialize domains report: %w", err)
	}
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      req.Params.URI,
			MIMEType: "application/json",
			Text:     string(jsonData),
		},
	}, nil
}

func (p *DomainServerPlugin) buildListGlobalDomainsTool() mcp.Tool {
	return mcp.NewTool(
		"list_global_domains",
		mcp.WithDescription("List all global domains configured in Dokku"),
	)
}

func (p *DomainServerPlugin) handleListGlobalDomains(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domains, err := p.domainService.ListGlobalDomains(ctx)
	if err != nil {
		env := server.ToolResponse{Status: server.ToolStatusError, Code: "DOMAINS_LIST_FAILED", Message: fmt.Sprintf("Failed to list global domains: %v", err)}
		b, _ := json.MarshalIndent(env, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	}
	env := server.ToolResponse{Status: server.ToolStatusOK, Code: "DOMAINS_OK", Data: map[string]any{"domains": domains}}
	b, _ := json.MarshalIndent(env, "", "  ")
	return mcp.NewToolResultText(string(b)), nil
}

func (p *DomainServerPlugin) buildAddGlobalDomainTool() mcp.Tool {
	return mcp.NewTool(
		"add_global_domain",
		mcp.WithDescription("Add a global domain to Dokku"),
		mcp.WithString("domain_name",
			mcp.Required(),
			mcp.Description("The domain name to add"),
		),
	)
}

func (p *DomainServerPlugin) handleAddGlobalDomain(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domainName, err := req.RequireString("domain_name")
	if err != nil {
		return mcp.NewToolResultError("Domain name is required"), nil
	}

	if err := p.domainService.AddGlobalDomain(ctx, domainName); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add global domain: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("✅ Global domain '%s' added successfully", domainName)), nil
}

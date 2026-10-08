package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	dokkuApi "github.com/dokku-mcp/dokku-mcp/internal/dokku-api"
	"github.com/dokku-mcp/dokku-mcp/internal/server"
	serverDomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/domain/application"
	domaindomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/domain/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/domain/infrastructure"
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
		{
			Name:        "get_app_domains",
			Description: "List the domains of an application",
			Builder:     buildGetAppDomainsTool,
			Handler:     p.handleGetAppDomains,
		},
		{
			Name:        "add_app_domain",
			Description: "Add a domain to an application",
			Builder:     buildAddAppDomainTool,
			Handler:     p.handleAddAppDomain,
		},
		{
			Name:        "remove_app_domain",
			Description: "Remove a domain from an application",
			Builder:     buildRemoveAppDomainTool,
			Handler:     p.handleRemoveAppDomain,
		},
	}, nil
}

func buildGetAppDomainsTool() mcp.Tool {
	return mcp.NewTool(
		"get_app_domains",
		mcp.WithTitleAnnotation("Get application domains"),
		mcp.WithDescription("List the domains (vhosts) an application answers on, plus the global domains"),
		mcp.WithString("app_name", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithOutputSchema[domaindomain.AppDomains](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func buildAddAppDomainTool() mcp.Tool {
	return mcp.NewTool(
		"add_app_domain",
		mcp.WithTitleAnnotation("Add application domain"),
		mcp.WithDescription("Add a domain to an application. Point the domain's DNS at the Dokku host, "+
			"then use enable_letsencrypt for HTTPS"),
		mcp.WithString("app_name", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithString("domain_name", mcp.Required(), mcp.Description("Domain to add, e.g. www.example.com")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func buildRemoveAppDomainTool() mcp.Tool {
	return mcp.NewTool(
		"remove_app_domain",
		mcp.WithTitleAnnotation("Remove application domain"),
		mcp.WithDescription("Stop serving an application on a domain"),
		mcp.WithString("app_name", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithString("domain_name", mcp.Required(), mcp.Description("Domain to remove")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func (p *DomainServerPlugin) handleGetAppDomains(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appName, err := req.RequireString("app_name")
	if err != nil {
		return mcp.NewToolResultError("app_name is required"), nil
	}
	domains, err := p.domainService.GetAppDomains(ctx, appName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get domains: %v", err)), nil
	}
	return mcp.NewToolResultStructuredOnly(domains), nil
}

func (p *DomainServerPlugin) handleAddAppDomain(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appName, err := req.RequireString("app_name")
	if err != nil {
		return mcp.NewToolResultError("app_name is required"), nil
	}
	domainName, err := req.RequireString("domain_name")
	if err != nil {
		return mcp.NewToolResultError("domain_name is required"), nil
	}
	if err := p.domainService.AddAppDomain(ctx, appName, domainName); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to add domain: %v", err)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Domain '%s' added to '%s'", domainName, appName)), nil
}

func (p *DomainServerPlugin) handleRemoveAppDomain(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appName, err := req.RequireString("app_name")
	if err != nil {
		return mcp.NewToolResultError("app_name is required"), nil
	}
	domainName, err := req.RequireString("domain_name")
	if err != nil {
		return mcp.NewToolResultError("domain_name is required"), nil
	}
	if err := p.domainService.RemoveAppDomain(ctx, appName, domainName); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to remove domain: %v", err)), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Domain '%s' removed from '%s'", domainName, appName)), nil
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
		mcp.WithTitleAnnotation("List global domains"),
		mcp.WithDescription("List all global domains configured in Dokku"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func (p *DomainServerPlugin) handleListGlobalDomains(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	domains, err := p.domainService.ListGlobalDomains(ctx)
	if err != nil {
		env := server.ToolResponse{Status: server.ToolStatusError, Code: "DOMAINS_LIST_FAILED", Message: fmt.Sprintf("Failed to list global domains: %v", err)}
		b, _ := json.MarshalIndent(env, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	}
	payload, err := json.Marshal(domains)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to encode domains: %v", err)), nil
	}

	env := server.ToolResponse{Status: server.ToolStatusOK, Code: "DOMAINS_OK", Data: server.ToolResponseData{"domains": payload}}
	b, _ := json.MarshalIndent(env, "", "  ")
	return mcp.NewToolResultText(string(b)), nil
}

func (p *DomainServerPlugin) buildAddGlobalDomainTool() mcp.Tool {
	return mcp.NewTool(
		"add_global_domain",
		mcp.WithTitleAnnotation("Add global domain"),
		mcp.WithDescription("Add a global domain to Dokku"),
		mcp.WithString("domain_name",
			mcp.Required(),
			mcp.Description("The domain name to add"),
		),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
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

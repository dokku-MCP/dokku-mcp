// Package services exposes Dokku's official datastore plugins (postgres,
// mysql, redis, ...) through a single set of type-parameterised tools.
package services

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	dokkuApi "github.com/dokku-mcp/dokku-mcp/internal/dokku-api"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/infrastructure"
	appdomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/app/domain"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/fx"
)

// ServicesServerPlugin provides datastore management tools.
type ServicesServerPlugin struct {
	adapter *Adapter
	logger  *slog.Logger
}

// NewServicesServerPlugin creates the datastore plugin.
func NewServicesServerPlugin(adapter *Adapter, logger *slog.Logger) domain.ServerPlugin {
	return &ServicesServerPlugin{adapter: adapter, logger: logger}
}

func (p *ServicesServerPlugin) ID() string   { return "services" }
func (p *ServicesServerPlugin) Name() string { return "Dokku Datastores" }
func (p *ServicesServerPlugin) Description() string {
	return "Create, link and inspect datastores (postgres, mysql, redis, ...) provided by Dokku's official plugins"
}
func (p *ServicesServerPlugin) Version() string { return "0.1.0" }

// DokkuPluginName is empty: the plugin stays active and each tool checks
// that the requested datastore plugin is installed, so the model gets an
// actionable message instead of a missing tool.
func (p *ServicesServerPlugin) DokkuPluginName() string { return "" }

// ServiceList is the structured result of list_services.
type ServiceList struct {
	InstalledTypes []string            `json:"installed_types" jsonschema:"Datastore plugins installed on the server"`
	Services       map[string][]string `json:"services" jsonschema:"Service names by datastore type"`
}

// ServiceInfo is the structured result of get_service_info.
type ServiceInfo struct {
	ServiceType string            `json:"service_type"`
	Name        string            `json:"name"`
	Info        map[string]string `json:"info" jsonschema:"Service report; passwords are masked"`
	LinkedApps  []string          `json:"linked_apps"`
}

func serviceTypeParam() mcp.ToolOption {
	return mcp.WithString("service_type",
		mcp.Required(),
		mcp.Description("Datastore type"),
		mcp.Enum(supportedTypes...),
	)
}

func serviceNameParam(description string) mcp.ToolOption {
	return mcp.WithString("name",
		mcp.Required(),
		mcp.Description(description),
		mcp.Pattern(serviceNamePattern.String()),
	)
}

func (p *ServicesServerPlugin) GetTools(ctx context.Context) ([]domain.Tool, error) {
	return []domain.Tool{
		{Name: "list_services", Description: "List datastore services", Builder: buildListServicesTool, Handler: p.handleListServices},
		{Name: "create_service", Description: "Create a datastore service", Builder: buildCreateServiceTool, Handler: p.handleCreateService},
		{Name: "get_service_info", Description: "Get datastore service information", Builder: buildGetServiceInfoTool, Handler: p.handleGetServiceInfo},
		{Name: "link_service", Description: "Link a datastore to an application", Builder: buildLinkServiceTool, Handler: p.handleLinkService},
		{Name: "unlink_service", Description: "Unlink a datastore from an application", Builder: buildUnlinkServiceTool, Handler: p.handleUnlinkService},
		{Name: "get_service_logs", Description: "Get datastore container logs", Builder: buildGetServiceLogsTool, Handler: p.handleGetServiceLogs},
		{Name: "destroy_service", Description: "Destroy a datastore service", Builder: buildDestroyServiceTool, Handler: p.handleDestroyService},
	}, nil
}

func buildListServicesTool() mcp.Tool {
	return mcp.NewTool("list_services",
		mcp.WithTitleAnnotation("List datastores"),
		mcp.WithDescription("List datastore services (postgres, redis, ...) and which datastore plugins are installed"),
		mcp.WithString("service_type",
			mcp.Description("Only list services of this type"),
			mcp.Enum(supportedTypes...),
		),
		mcp.WithOutputSchema[ServiceList](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func buildCreateServiceTool() mcp.Tool {
	return mcp.NewTool("create_service",
		mcp.WithTitleAnnotation("Create datastore"),
		mcp.WithDescription("Create a datastore service. Link it to an app with link_service to inject its connection URL"),
		serviceTypeParam(),
		serviceNameParam("Name of the new service"),
		mcp.WithString("image_version",
			mcp.Description("Optional image tag, e.g. \"17\" for postgres:17"),
		),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func buildGetServiceInfoTool() mcp.Tool {
	return mcp.NewTool("get_service_info",
		mcp.WithTitleAnnotation("Get datastore info"),
		mcp.WithDescription("Get a datastore's status, version and linked applications. Credentials are masked"),
		serviceTypeParam(),
		serviceNameParam("Name of the service"),
		mcp.WithOutputSchema[ServiceInfo](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func buildLinkServiceTool() mcp.Tool {
	return mcp.NewTool("link_service",
		mcp.WithTitleAnnotation("Link datastore to app"),
		mcp.WithDescription("Link a datastore to an application. Dokku sets <ALIAS>_URL on the app "+
			"(DATABASE_URL, REDIS_URL, ... by default) and restarts it unless no_restart is true"),
		serviceTypeParam(),
		serviceNameParam("Name of the service"),
		mcp.WithString("app_name", mcp.Required(), mcp.Description("Application to link")),
		mcp.WithString("alias",
			mcp.Description("Environment variable prefix, e.g. BLUE_DATABASE sets BLUE_DATABASE_URL"),
			mcp.Pattern(aliasPattern.String()),
		),
		mcp.WithBoolean("no_restart", mcp.Description("Do not restart the application")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func buildUnlinkServiceTool() mcp.Tool {
	return mcp.NewTool("unlink_service",
		mcp.WithTitleAnnotation("Unlink datastore from app"),
		mcp.WithDescription("Unlink a datastore from an application, removing its connection URL from the app"),
		serviceTypeParam(),
		serviceNameParam("Name of the service"),
		mcp.WithString("app_name", mcp.Required(), mcp.Description("Application to unlink")),
		mcp.WithBoolean("no_restart", mcp.Description("Do not restart the application")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func buildGetServiceLogsTool() mcp.Tool {
	return mcp.NewTool("get_service_logs",
		mcp.WithTitleAnnotation("Get datastore logs"),
		mcp.WithDescription("Get the last 100 log lines of a datastore container"),
		serviceTypeParam(),
		serviceNameParam("Name of the service"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func buildDestroyServiceTool() mcp.Tool {
	return mcp.NewTool("destroy_service",
		mcp.WithTitleAnnotation("Destroy datastore"),
		mcp.WithDescription("Permanently delete a datastore and ALL of its data. Unlink it from every app first. "+
			"Only call this when the user explicitly asked for it"),
		serviceTypeParam(),
		serviceNameParam("Name of the service"),
		mcp.WithString("confirm_name",
			mcp.Required(),
			mcp.Description("Repeat the service name to confirm the deletion"),
		),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

// serviceArgs reads and validates the type and name shared by most tools.
func serviceArgs(req mcp.CallToolRequest) (string, string, error) {
	serviceType, err := req.RequireString("service_type")
	if err != nil {
		return "", "", fmt.Errorf("service_type is required")
	}
	if err := ValidateServiceType(serviceType); err != nil {
		return "", "", err
	}
	name, err := req.RequireString("name")
	if err != nil {
		return "", "", fmt.Errorf("name is required")
	}
	if err := ValidateServiceName(name); err != nil {
		return "", "", err
	}
	return serviceType, name, nil
}

func appNameArg(req mcp.CallToolRequest) (string, error) {
	appName, err := req.RequireString("app_name")
	if err != nil {
		return "", fmt.Errorf("app_name is required")
	}
	normalized, err := appdomain.NewApplicationName(appName)
	if err != nil {
		return "", err
	}
	return normalized.Value(), nil
}

func textResult(summary, output string) *mcp.CallToolResult {
	if output = strings.TrimSpace(output); output != "" {
		summary += "\n\n" + output
	}
	return mcp.NewToolResultText(summary)
}

func (p *ServicesServerPlugin) handleListServices(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	installed, err := p.adapter.InstalledTypes(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	types := installed
	if only := req.GetString("service_type", ""); only != "" {
		if err := p.adapter.requireInstalled(ctx, only); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		types = []string{only}
	}

	result := ServiceList{InstalledTypes: installed, Services: map[string][]string{}}
	if result.InstalledTypes == nil {
		result.InstalledTypes = []string{}
	}
	for _, t := range types {
		names, err := p.adapter.List(ctx, t)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		result.Services[t] = names
	}
	return mcp.NewToolResultStructuredOnly(result), nil
}

func (p *ServicesServerPlugin) handleCreateService(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceType, name, err := serviceArgs(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	out, err := p.adapter.Create(ctx, serviceType, name, CreateOptions{ImageVersion: req.GetString("image_version", "")})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return textResult(fmt.Sprintf("Created %s service '%s'. Link it to an app with link_service.", serviceType, name), RedactCredentials(out)), nil
}

func (p *ServicesServerPlugin) handleGetServiceInfo(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceType, name, err := serviceArgs(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	info, err := p.adapter.Info(ctx, serviceType, name)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	links, err := p.adapter.Links(ctx, serviceType, name)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultStructuredOnly(ServiceInfo{ServiceType: serviceType, Name: name, Info: info, LinkedApps: links}), nil
}

func (p *ServicesServerPlugin) handleLinkService(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceType, name, err := serviceArgs(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	appName, err := appNameArg(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	opts := LinkOptions{Alias: req.GetString("alias", ""), NoRestart: req.GetBool("no_restart", false)}
	out, err := p.adapter.Link(ctx, serviceType, name, appName, opts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return textResult(fmt.Sprintf("Linked %s service '%s' to '%s'.", serviceType, name, appName), RedactCredentials(out)), nil
}

func (p *ServicesServerPlugin) handleUnlinkService(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceType, name, err := serviceArgs(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	appName, err := appNameArg(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	out, err := p.adapter.Unlink(ctx, serviceType, name, appName, req.GetBool("no_restart", false))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return textResult(fmt.Sprintf("Unlinked %s service '%s' from '%s'.", serviceType, name, appName), out), nil
}

func (p *ServicesServerPlugin) handleGetServiceLogs(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceType, name, err := serviceArgs(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	out, err := p.adapter.Logs(ctx, serviceType, name)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(RedactCredentials(out)), nil
}

func (p *ServicesServerPlugin) handleDestroyService(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceType, name, err := serviceArgs(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if req.GetString("confirm_name", "") != name {
		return mcp.NewToolResultError("confirm_name must exactly match the service name"), nil
	}
	out, err := p.adapter.Destroy(ctx, serviceType, name)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return textResult(fmt.Sprintf("Destroyed %s service '%s'.", serviceType, name), out), nil
}

var Module = fx.Module("services",
	fx.Provide(
		func(client dokkuApi.DokkuClient, logger *slog.Logger) *Adapter {
			return NewAdapter(client, infrastructure.NewPluginDiscoveryService(client, logger))
		},
		fx.Annotate(
			NewServicesServerPlugin,
			fx.As(new(domain.ServerPlugin)),
			fx.ResultTags(`group:"server_plugins"`),
		),
	),
)

// Package letsencrypt exposes the dokku-letsencrypt plugin: HTTPS
// certificates for applications, with automatic renewal.
package letsencrypt

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	dokkuApi "github.com/dokku-mcp/dokku-mcp/internal/dokku-api"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	appdomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/app/domain"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/fx"
)

var emailPattern = regexp.MustCompile(`^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`)

// LetsEncryptServerPlugin manages HTTPS certificates. It is only active when
// the letsencrypt Dokku plugin is installed.
type LetsEncryptServerPlugin struct {
	client dokkuApi.DokkuClient
	logger *slog.Logger
}

// NewLetsEncryptServerPlugin creates the plugin.
func NewLetsEncryptServerPlugin(client dokkuApi.DokkuClient, logger *slog.Logger) domain.ServerPlugin {
	return &LetsEncryptServerPlugin{client: client, logger: logger}
}

func (p *LetsEncryptServerPlugin) ID() string   { return "letsencrypt" }
func (p *LetsEncryptServerPlugin) Name() string { return "Dokku Let's Encrypt" }
func (p *LetsEncryptServerPlugin) Description() string {
	return "HTTPS certificates from Let's Encrypt for Dokku applications"
}
func (p *LetsEncryptServerPlugin) Version() string         { return "0.1.0" }
func (p *LetsEncryptServerPlugin) DokkuPluginName() string { return "letsencrypt" }

// Status is the structured result of get_letsencrypt_status.
type Status struct {
	AppName string            `json:"app_name"`
	Active  bool              `json:"active" jsonschema:"Whether the app serves a Let's Encrypt certificate"`
	Report  map[string]string `json:"report" jsonschema:"letsencrypt:report fields"`
}

func (p *LetsEncryptServerPlugin) GetTools(ctx context.Context) ([]domain.Tool, error) {
	return []domain.Tool{
		{Name: "get_letsencrypt_status", Description: "Get HTTPS certificate status", Builder: buildStatusTool, Handler: p.handleStatus},
		{Name: "enable_letsencrypt", Description: "Enable HTTPS for an application", Builder: buildEnableTool, Handler: p.handleEnable},
		{Name: "disable_letsencrypt", Description: "Disable HTTPS for an application", Builder: buildDisableTool, Handler: p.handleDisable},
		{Name: "set_letsencrypt_email", Description: "Set the ACME account email", Builder: buildSetEmailTool, Handler: p.handleSetEmail},
	}, nil
}

func buildStatusTool() mcp.Tool {
	return mcp.NewTool("get_letsencrypt_status",
		mcp.WithTitleAnnotation("Get HTTPS certificate status"),
		mcp.WithDescription("Report whether an application has a Let's Encrypt certificate, its email and expiry"),
		mcp.WithString("app_name", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithOutputSchema[Status](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func buildEnableTool() mcp.Tool {
	return mcp.NewTool("enable_letsencrypt",
		mcp.WithTitleAnnotation("Enable HTTPS"),
		mcp.WithDescription("Request a Let's Encrypt certificate for every domain of an application and serve it over HTTPS. "+
			"The domains must already resolve to the Dokku host and the app must be deployed. "+
			"An ACME email is required: pass email or set one with set_letsencrypt_email"),
		mcp.WithString("app_name", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithString("email", mcp.Description("ACME account email for this application")),
		mcp.WithBoolean("auto_renew",
			mcp.Description("Install the daily renewal cron job (default true)"),
			mcp.DefaultBool(true),
		),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

func buildDisableTool() mcp.Tool {
	return mcp.NewTool("disable_letsencrypt",
		mcp.WithTitleAnnotation("Disable HTTPS"),
		mcp.WithDescription("Remove an application's Let's Encrypt certificate; it will be served over plain HTTP"),
		mcp.WithString("app_name", mcp.Required(), mcp.Description("Name of the application")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func buildSetEmailTool() mcp.Tool {
	return mcp.NewTool("set_letsencrypt_email",
		mcp.WithTitleAnnotation("Set ACME email"),
		mcp.WithDescription("Set the email used to register with Let's Encrypt, for one application or globally"),
		mcp.WithString("email", mcp.Required(), mcp.Description("Email address")),
		mcp.WithString("app_name", mcp.Description("Application to configure; omit to set the global default")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func requireAppName(req mcp.CallToolRequest) (string, error) {
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

func validateEmail(email string) error {
	if !emailPattern.MatchString(email) {
		return fmt.Errorf("invalid email address %q", email)
	}
	return nil
}

func (p *LetsEncryptServerPlugin) run(ctx context.Context, command string, args ...string) (string, error) {
	out, err := p.client.ExecuteCommand(ctx, command, args)
	if err != nil {
		return "", fmt.Errorf("%s failed: %w", command, err)
	}
	return string(out), nil
}

func (p *LetsEncryptServerPlugin) handleStatus(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appName, err := requireAppName(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	out, err := p.run(ctx, "letsencrypt:report", appName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	status := Status{AppName: appName, Report: map[string]string{}}
	for line := range strings.Lines(out) {
		if strings.HasPrefix(strings.TrimSpace(line), "=") {
			continue
		}
		if key, value, ok := dokkuApi.ParseColonKeyValueLine(line); ok && key != "" {
			status.Report[key] = value
		}
	}
	active, err := p.run(ctx, "letsencrypt:active", appName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	status.Active = strings.TrimSpace(active) == "true"
	return mcp.NewToolResultStructuredOnly(status), nil
}

func (p *LetsEncryptServerPlugin) handleEnable(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appName, err := requireAppName(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if email := req.GetString("email", ""); email != "" {
		if err := validateEmail(email); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		if _, err := p.run(ctx, "letsencrypt:set", appName, "email", email); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
	}

	out, err := p.run(ctx, "letsencrypt:enable", appName)
	if err != nil {
		return mcp.NewToolResultError(err.Error() +
			"\nCheck that an ACME email is set, the app is deployed and its domains resolve to this server."), nil
	}

	summary := fmt.Sprintf("HTTPS enabled for '%s'.", appName)
	if req.GetBool("auto_renew", true) {
		if _, err := p.run(ctx, "letsencrypt:cron-job", "--add"); err != nil {
			p.logger.Warn("Failed to install letsencrypt renewal cron job", "error", err)
			summary += " Warning: the renewal cron job could not be installed: " + err.Error()
		} else {
			summary += " Automatic renewal is enabled."
		}
	}
	return mcp.NewToolResultText(summary + "\n\n" + strings.TrimSpace(out)), nil
}

func (p *LetsEncryptServerPlugin) handleDisable(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appName, err := requireAppName(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if _, err := p.run(ctx, "letsencrypt:disable", appName); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("HTTPS certificate removed for '%s'.", appName)), nil
}

func (p *LetsEncryptServerPlugin) handleSetEmail(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	email, err := req.RequireString("email")
	if err != nil {
		return mcp.NewToolResultError("email is required"), nil
	}
	if err := validateEmail(email); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	target, scope := "--global", "globally"
	if req.GetString("app_name", "") != "" {
		appName, err := requireAppName(req)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		target, scope = appName, "for '"+appName+"'"
	}
	if _, err := p.run(ctx, "letsencrypt:set", target, "email", email); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Let's Encrypt email set %s.", scope)), nil
}

var Module = fx.Module("letsencrypt",
	fx.Provide(
		fx.Annotate(
			NewLetsEncryptServerPlugin,
			fx.As(new(domain.ServerPlugin)),
			fx.ResultTags(`group:"server_plugins"`),
		),
	),
)

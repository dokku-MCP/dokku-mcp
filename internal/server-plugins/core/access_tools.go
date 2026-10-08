package core

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	serverDomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	appdomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/app/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/core/domain"
	"github.com/mark3labs/mcp-go/mcp"
)

var (
	sshKeyNameRegex = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)
	// publicKeyRegex accepts a single OpenSSH public key line, including
	// certificate keys (*-cert-v01@openssh.com) issued by an SSH CA.
	publicKeyRegex = regexp.MustCompile(`^(` +
		`(ssh-(rsa|ed25519|dss)|ecdsa-sha2-nistp(256|384|521)|sk-(ssh-ed25519|ecdsa-sha2-nistp256)@openssh\.com)` +
		`|(ssh-(rsa|ed25519|dss)|rsa-sha2-(256|512)|ecdsa-sha2-nistp(256|384|521)|sk-(ssh-ed25519|ecdsa-sha2-nistp256))-cert-v01@openssh\.com` +
		`) [A-Za-z0-9+/]+={0,3}( [^\r\n]*)?$`)
	registryHostRegex = regexp.MustCompile(`^[A-Za-z0-9.-]+(:[0-9]+)?$`)
)

// SSHKeyList is the structured result of list_ssh_keys.
type SSHKeyList struct {
	Keys []domain.SSHKey `json:"keys"`
}

// RegistryReport is the structured result of get_registry_report.
type RegistryReport struct {
	AppName string            `json:"app_name,omitempty" jsonschema:"Empty for the global report"`
	Report  map[string]string `json:"report"`
}

func (p *CoreServerPlugin) accessTools() []serverDomain.Tool {
	return []serverDomain.Tool{
		{Name: "list_ssh_keys", Description: "List SSH keys allowed to push to Dokku", Builder: buildListSSHKeysTool, Handler: p.handleListSSHKeys},
		{Name: "add_ssh_key", Description: "Allow a public key to push to Dokku", Builder: buildAddSSHKeyTool, Handler: p.handleAddSSHKey},
		{Name: "remove_ssh_key", Description: "Revoke an SSH key", Builder: buildRemoveSSHKeyTool, Handler: p.handleRemoveSSHKey},
		{Name: "get_registry_report", Description: "Get Docker registry settings", Builder: buildRegistryReportTool, Handler: p.handleRegistryReport},
		{Name: "registry_login", Description: "Log in to a Docker registry", Builder: buildRegistryLoginTool, Handler: p.handleRegistryLogin},
		{Name: "registry_logout", Description: "Log out of a Docker registry", Builder: buildRegistryLogoutTool, Handler: p.handleRegistryLogout},
	}
}

func buildListSSHKeysTool() mcp.Tool {
	return mcp.NewTool("list_ssh_keys",
		mcp.WithTitleAnnotation("List SSH keys"),
		mcp.WithDescription("List the SSH keys (name and fingerprint) allowed to deploy to this Dokku server"),
		mcp.WithOutputSchema[SSHKeyList](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func buildAddSSHKeyTool() mcp.Tool {
	return mcp.NewTool("add_ssh_key",
		mcp.WithTitleAnnotation("Add SSH key"),
		mcp.WithDescription("Grant a public key push access to every app on this Dokku server. "+
			"Only call this when the user explicitly asked to give someone access"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Unique key name, e.g. the person's username"),
			mcp.Pattern(sshKeyNameRegex.String())),
		mcp.WithString("public_key", mcp.Required(), mcp.Description("OpenSSH public key line, e.g. \"ssh-ed25519 AAAA... user@host\"")),
		mcp.WithBoolean("allow_admin", mcp.Description(
			"Allow a name containing \"admin\". Dokku lets such keys add further keys remotely")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func buildRemoveSSHKeyTool() mcp.Tool {
	return mcp.NewTool("remove_ssh_key",
		mcp.WithTitleAnnotation("Remove SSH key"),
		mcp.WithDescription("Revoke an SSH key's access to this Dokku server"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Key name from list_ssh_keys")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func buildRegistryReportTool() mcp.Tool {
	return mcp.NewTool("get_registry_report",
		mcp.WithTitleAnnotation("Get registry settings"),
		mcp.WithDescription("Show Docker registry settings (server, image repository, push-on-release) for an app or globally"),
		mcp.WithString("app_name", mcp.Description("Application; omit for the global settings")),
		mcp.WithOutputSchema[RegistryReport](),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func buildRegistryLoginTool() mcp.Tool {
	return mcp.NewTool("registry_login",
		mcp.WithTitleAnnotation("Log in to registry"),
		mcp.WithDescription("Store Docker registry credentials on the Dokku server, for one app or globally. "+
			"Prefer a scoped access token over an account password; it is sent over stdin, never on the command line"),
		mcp.WithString("server", mcp.Required(), mcp.Description("Registry host, e.g. ghcr.io")),
		mcp.WithString("username", mcp.Required(), mcp.Description("Registry username")),
		mcp.WithString("password", mcp.Required(), mcp.Description("Password or access token")),
		mcp.WithString("app_name", mcp.Description("Application; omit to log in globally")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

func buildRegistryLogoutTool() mcp.Tool {
	return mcp.NewTool("registry_logout",
		mcp.WithTitleAnnotation("Log out of registry"),
		mcp.WithDescription("Remove stored Docker registry credentials, for one app or globally"),
		mcp.WithString("server", mcp.Required(), mcp.Description("Registry host, e.g. ghcr.io")),
		mcp.WithString("app_name", mcp.Description("Application; omit to log out globally")),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(false),
	)
}

func validateSSHKeyName(name string, allowAdmin bool) error {
	if !sshKeyNameRegex.MatchString(name) {
		return fmt.Errorf("invalid key name %q: use letters, digits, '.', '_' and '-' (max 64 characters)", name)
	}
	if !allowAdmin && strings.Contains(strings.ToLower(name), "admin") {
		return fmt.Errorf("key names containing \"admin\" can add further keys remotely; set allow_admin to true only if the user explicitly wants that")
	}
	return nil
}

// optionalAppName returns the app_name argument, validated, or "" when absent.
func optionalAppName(req mcp.CallToolRequest) (string, error) {
	appName := req.GetString("app_name", "")
	if appName == "" {
		return "", nil
	}
	normalized, err := appdomain.NewApplicationName(appName)
	if err != nil {
		return "", err
	}
	return normalized.Value(), nil
}

func registryServerArg(req mcp.CallToolRequest) (string, error) {
	server, err := req.RequireString("server")
	if err != nil {
		return "", fmt.Errorf("server is required")
	}
	if !registryHostRegex.MatchString(server) {
		return "", fmt.Errorf("invalid registry host %q", server)
	}
	return server, nil
}

func (p *CoreServerPlugin) handleListSSHKeys(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keys, err := p.coreService.ListSSHKeys(ctx)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultStructuredOnly(SSHKeyList{Keys: keys}), nil
}

func (p *CoreServerPlugin) handleAddSSHKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("name is required"), nil
	}
	if err := validateSSHKeyName(name, req.GetBool("allow_admin", false)); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	key := strings.TrimSpace(req.GetString("public_key", ""))
	if !publicKeyRegex.MatchString(key) {
		return mcp.NewToolResultError("public_key must be a single OpenSSH public key line (ssh-ed25519, ssh-rsa, ecdsa-sha2-*, sk-*); never a private key"), nil
	}
	if err := p.coreService.AddSSHKey(ctx, name, key); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("SSH key '%s' added; it can now push to Dokku.", name)), nil
}

func (p *CoreServerPlugin) handleRemoveSSHKey(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("name is required"), nil
	}
	if err := validateSSHKeyName(name, true); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := p.coreService.RemoveSSHKey(ctx, name); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("SSH key '%s' removed.", name)), nil
}

func (p *CoreServerPlugin) handleRegistryReport(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	appName, err := optionalAppName(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	report, err := p.coreService.GetRegistryReport(ctx, appName)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultStructuredOnly(RegistryReport{AppName: appName, Report: report}), nil
}

func (p *CoreServerPlugin) handleRegistryLogin(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	server, err := registryServerArg(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	appName, err := optionalAppName(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	username := req.GetString("username", "")
	password := req.GetString("password", "")
	if strings.ContainsAny(password, "\r\n") {
		return mcp.NewToolResultError("password must be a single line"), nil
	}
	if err := p.coreService.LoginRegistry(ctx, appName, server, username, password); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Logged in to %s as %s.", server, username)), nil
}

func (p *CoreServerPlugin) handleRegistryLogout(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	server, err := registryServerArg(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	appName, err := optionalAppName(req)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := p.coreService.LogoutRegistry(ctx, appName, server); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(fmt.Sprintf("Logged out of %s.", server)), nil
}

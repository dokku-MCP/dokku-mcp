package core

import (
	"io"
	"log/slog"
	"testing"

	"github.com/dokku-mcp/dokku-mcp/internal/dokku-api/dokkutest"
	serverDomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/plugintest"
	"github.com/dokku-mcp/dokku-mcp/pkg/config"
)

const testKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGd2Zm9vYmFyYmF6cXV4cXV1eGNvcmdlZ3JhdWx0 alice@laptop"

func newPlugin(t *testing.T) (*dokkutest.FakeClient, serverDomain.ToolProvider) {
	t.Helper()
	client := dokkutest.NewFakeClient()
	plugin := NewCoreServerPlugin(client, slog.New(slog.NewTextHandler(io.Discard, nil)), &config.ServerConfig{})
	return client, plugin.(serverDomain.ToolProvider)
}

func TestCoreToolsAreAnnotated(t *testing.T) {
	_, plugin := newPlugin(t)
	plugintest.RequireAnnotated(t, plugin)
}

func TestListSSHKeysParsesNames(t *testing.T) {
	client, plugin := newPlugin(t)
	client.Respond("ssh-keys:list", "SHA256:abc123 NAME=\"alice\" SSHCOMMAND_ALLOWED_KEYS=\"none\"\nSHA256:def456 NAME=\"admin\" SSHCOMMAND_ALLOWED_KEYS=\"none\"\n")

	list := plugintest.Structured[SSHKeyList](t, plugintest.CallTool(t, plugin, "list_ssh_keys", nil))
	if len(list.Keys) != 2 || list.Keys[0].Name != "alice" || list.Keys[0].Fingerprint != "SHA256:abc123" || list.Keys[1].Name != "admin" {
		t.Fatalf("unexpected keys: %+v", list.Keys)
	}
}

func TestAddSSHKeySendsKeyOnStdin(t *testing.T) {
	client, plugin := newPlugin(t)

	plugintest.RequireSuccess(t, plugintest.CallTool(t, plugin, "add_ssh_key", map[string]any{"name": "alice", "public_key": testKey}))

	calls := client.CallsTo("ssh-keys:add")
	if len(calls) != 1 || calls[0].String() != "ssh-keys:add alice" || calls[0].Stdin != testKey+"\n" {
		t.Fatalf("unexpected calls: %+v", calls)
	}
}

func TestAddSSHKeyGuardsAdminNamesAndPrivateKeys(t *testing.T) {
	client, plugin := newPlugin(t)

	plugintest.RequireError(t, plugintest.CallTool(t, plugin, "add_ssh_key", map[string]any{"name": "ci-admin", "public_key": testKey}), "allow_admin")
	plugintest.RequireError(t, plugintest.CallTool(t, plugin, "add_ssh_key", map[string]any{
		"name": "alice", "public_key": "-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAA\n-----END OPENSSH PRIVATE KEY-----",
	}), "never a private key")
	if len(client.CallsTo("ssh-keys:add")) != 0 {
		t.Fatal("rejected keys must not reach Dokku")
	}

	plugintest.RequireSuccess(t, plugintest.CallTool(t, plugin, "add_ssh_key", map[string]any{"name": "ci-admin", "public_key": testKey, "allow_admin": true}))
}

func TestRegistryLoginUsesPasswordStdin(t *testing.T) {
	client, plugin := newPlugin(t)

	plugintest.RequireSuccess(t, plugintest.CallTool(t, plugin, "registry_login", map[string]any{
		"server": "ghcr.io", "username": "acme-bot", "password": "ghp_s3cret value", "app_name": "web",
	}))
	plugintest.RequireSuccess(t, plugintest.CallTool(t, plugin, "registry_login", map[string]any{
		"server": "ghcr.io", "username": "acme-bot", "password": "tok",
	}))

	calls := client.CallsTo("registry:login")
	if len(calls) != 2 {
		t.Fatalf("unexpected calls: %+v", calls)
	}
	if calls[0].String() != "registry:login --password-stdin web ghcr.io acme-bot" || calls[0].Stdin != "ghp_s3cret value\n" {
		t.Fatalf("unexpected app login: %+v", calls[0])
	}
	if calls[1].String() != "registry:login --password-stdin --global ghcr.io acme-bot" {
		t.Fatalf("unexpected global login: %+v", calls[1])
	}
}

func TestRegistryLogoutAndReport(t *testing.T) {
	client, plugin := newPlugin(t)
	client.Respond("registry:report", "=====> web registry information\n       Registry image repo:   acme/web\n       Registry server:       ghcr.io\n")

	plugintest.RequireSuccess(t, plugintest.CallTool(t, plugin, "registry_logout", map[string]any{"server": "ghcr.io"}))
	if calls := client.CallsTo("registry:logout"); len(calls) != 1 || calls[0].String() != "registry:logout --global ghcr.io" {
		t.Fatalf("unexpected logout calls: %v", calls)
	}

	report := plugintest.Structured[RegistryReport](t, plugintest.CallTool(t, plugin, "get_registry_report", map[string]any{"app_name": "web"}))
	if report.Report["Registry server"] != "ghcr.io" || report.Report["Registry image repo"] != "acme/web" {
		t.Fatalf("unexpected report: %+v", report)
	}
}

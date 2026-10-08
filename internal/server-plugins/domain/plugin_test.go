package domain

import (
	"io"
	"log/slog"
	"slices"
	"testing"

	"github.com/dokku-mcp/dokku-mcp/internal/dokku-api/dokkutest"
	serverDomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/plugintest"
	domaindomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/domain/domain"
)

func newPlugin(t *testing.T) (*dokkutest.FakeClient, serverDomain.ToolProvider) {
	t.Helper()
	client := dokkutest.NewFakeClient()
	plugin := NewDomainServerPlugin(client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return client, plugin.(serverDomain.ToolProvider)
}

func TestDomainToolsAreAnnotated(t *testing.T) {
	_, plugin := newPlugin(t)
	plugintest.RequireAnnotated(t, plugin)
}

func TestGetAppDomains(t *testing.T) {
	client, plugin := newPlugin(t)
	client.Respond("domains:report", `=====> web domains information
       Domains app enabled:           true
       Domains app vhosts:            web.example.com www.example.com
       Domains global enabled:        true
       Domains global vhosts:         example.com
`)

	result := plugintest.Structured[domaindomain.AppDomains](t, plugintest.CallTool(t, plugin, "get_app_domains", map[string]any{"app_name": "web"}))
	if !result.Enabled || !slices.Equal(result.Domains, []string{"web.example.com", "www.example.com"}) || !slices.Equal(result.GlobalDomains, []string{"example.com"}) {
		t.Fatalf("unexpected domains: %+v", result)
	}
	if calls := client.CallsTo("domains:report"); len(calls) != 1 || calls[0].String() != "domains:report web" {
		t.Fatalf("unexpected calls: %v", calls)
	}
}

func TestAddAndRemoveAppDomain(t *testing.T) {
	client, plugin := newPlugin(t)
	plugintest.RequireSuccess(t, plugintest.CallTool(t, plugin, "add_app_domain", map[string]any{"app_name": "web", "domain_name": "www.example.com"}))
	plugintest.RequireSuccess(t, plugintest.CallTool(t, plugin, "remove_app_domain", map[string]any{"app_name": "web", "domain_name": "old.example.com"}))

	if calls := client.CallsTo("domains:add"); len(calls) != 1 || calls[0].String() != "domains:add web www.example.com" {
		t.Fatalf("unexpected add calls: %v", calls)
	}
	if calls := client.CallsTo("domains:remove"); len(calls) != 1 || calls[0].String() != "domains:remove web old.example.com" {
		t.Fatalf("unexpected remove calls: %v", calls)
	}
}

func TestAddAppDomainRejectsInvalidDomain(t *testing.T) {
	client, plugin := newPlugin(t)
	result := plugintest.CallTool(t, plugin, "add_app_domain", map[string]any{"app_name": "web", "domain_name": "not a domain"})
	plugintest.RequireError(t, result, "invalid domain")
	if len(client.Calls()) != 0 {
		t.Fatalf("expected no calls, got %v", client.Calls())
	}
}

package letsencrypt

import (
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/dokku-mcp/dokku-mcp/internal/dokku-api/dokkutest"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/plugintest"
)

func newPlugin(t *testing.T) (*dokkutest.FakeClient, domain.ToolProvider) {
	t.Helper()
	client := dokkutest.NewFakeClient()
	plugin := NewLetsEncryptServerPlugin(client, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return client, plugin.(domain.ToolProvider)
}

func commands(client *dokkutest.FakeClient) []string {
	var out []string
	for _, c := range client.Calls() {
		out = append(out, c.String())
	}
	return out
}

func TestLetsEncryptToolsAreAnnotated(t *testing.T) {
	_, plugin := newPlugin(t)
	plugintest.RequireAnnotated(t, plugin)
}

func TestEnableWithEmailAndRenewal(t *testing.T) {
	client, plugin := newPlugin(t)

	result := plugintest.CallTool(t, plugin, "enable_letsencrypt", plugintest.Args{
		"app_name": "web", "email": "ops@example.com",
	})
	plugintest.RequireSuccess(t, result)

	want := []string{
		"letsencrypt:set web email ops@example.com",
		"letsencrypt:enable web",
		"letsencrypt:cron-job --add",
	}
	if got := commands(client); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("commands = %v, want %v", got, want)
	}
}

func TestEnableWithoutRenewal(t *testing.T) {
	client, plugin := newPlugin(t)
	plugintest.RequireSuccess(t, plugintest.CallTool(t, plugin, "enable_letsencrypt", plugintest.Args{
		"app_name": "web", "auto_renew": false,
	}))
	if got := commands(client); len(got) != 1 || got[0] != "letsencrypt:enable web" {
		t.Fatalf("commands = %v", got)
	}
}

func TestEnableFailureExplainsPrerequisites(t *testing.T) {
	client, plugin := newPlugin(t)
	client.Fail("letsencrypt:enable", errors.New("exit status 1"))

	result := plugintest.CallTool(t, plugin, "enable_letsencrypt", plugintest.Args{"app_name": "web"})
	plugintest.RequireError(t, result, "domains resolve to this server")
}

func TestEnableRejectsInvalidEmail(t *testing.T) {
	client, plugin := newPlugin(t)
	result := plugintest.CallTool(t, plugin, "enable_letsencrypt", plugintest.Args{"app_name": "web", "email": "not-an-email"})
	plugintest.RequireError(t, result, "invalid email")
	if len(client.Calls()) != 0 {
		t.Fatalf("expected no calls, got %v", commands(client))
	}
}

func TestSetEmailGlobalAndPerApp(t *testing.T) {
	client, plugin := newPlugin(t)
	plugintest.RequireSuccess(t, plugintest.CallTool(t, plugin, "set_letsencrypt_email", plugintest.Args{"email": "a@example.com"}))
	plugintest.RequireSuccess(t, plugintest.CallTool(t, plugin, "set_letsencrypt_email", plugintest.Args{"email": "b@example.com", "app_name": "web"}))

	want := "letsencrypt:set --global email a@example.com|letsencrypt:set web email b@example.com"
	if got := strings.Join(commands(client), "|"); got != want {
		t.Fatalf("commands = %s", got)
	}
}

func TestStatus(t *testing.T) {
	client, plugin := newPlugin(t)
	client.Respond("letsencrypt:report", "=====> web letsencrypt information\n       Letsencrypt active:      true\n       Letsencrypt email:       ops@example.com\n")
	client.Respond("letsencrypt:active", "true\n")

	status := plugintest.Structured[Status](t, plugintest.CallTool(t, plugin, "get_letsencrypt_status", plugintest.Args{"app_name": "web"}))
	if !status.Active || status.Report["Letsencrypt email"] != "ops@example.com" {
		t.Fatalf("unexpected status: %+v", status)
	}
}

func TestEnableReportsRenewalFailure(t *testing.T) {
	client, plugin := newPlugin(t)
	client.Fail("letsencrypt:cron-job", errors.New("exit status 1"))

	result := plugintest.CallTool(t, plugin, "enable_letsencrypt", plugintest.Args{"app_name": "web"})
	plugintest.RequireSuccess(t, result)
	if !strings.Contains(plugintest.Text(result), "Warning: the renewal cron job could not be installed") {
		t.Fatalf("expected a renewal warning, got %s", plugintest.Text(result))
	}
}

func TestDisable(t *testing.T) {
	client, plugin := newPlugin(t)
	plugintest.RequireSuccess(t, plugintest.CallTool(t, plugin, "disable_letsencrypt", plugintest.Args{"app_name": "web"}))
	if got := commands(client); len(got) != 1 || got[0] != "letsencrypt:disable web" {
		t.Fatalf("commands = %v", got)
	}

	client.Fail("letsencrypt:disable", errors.New("exit status 1"))
	plugintest.RequireError(t, plugintest.CallTool(t, plugin, "disable_letsencrypt", plugintest.Args{"app_name": "web"}), "letsencrypt:disable failed")
}

func TestStatusFailsWhenActiveCheckFails(t *testing.T) {
	client, plugin := newPlugin(t)
	client.Fail("letsencrypt:active", errors.New("exit status 1"))
	plugintest.RequireError(t, plugintest.CallTool(t, plugin, "get_letsencrypt_status", plugintest.Args{"app_name": "web"}), "letsencrypt:active failed")
}

func TestAppNamesAreNormalized(t *testing.T) {
	client, plugin := newPlugin(t)
	plugintest.RequireSuccess(t, plugintest.CallTool(t, plugin, "disable_letsencrypt", plugintest.Args{"app_name": " WEB "}))
	if got := commands(client); len(got) != 1 || got[0] != "letsencrypt:disable web" {
		t.Fatalf("commands = %v", got)
	}
}

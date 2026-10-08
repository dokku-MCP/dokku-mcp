package services

import (
	"context"
	"io"
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/dokku-mcp/dokku-mcp/internal/dokku-api/dokkutest"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/plugintest"
)

type staticPlugins []string

func (s staticPlugins) GetEnabledDokkuPlugins(context.Context) ([]string, error) { return s, nil }

func newPlugin(t *testing.T, installed ...string) (*dokkutest.FakeClient, domain.ToolProvider) {
	t.Helper()
	client := dokkutest.NewFakeClient()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	plugin := NewServicesServerPlugin(NewAdapter(client, staticPlugins(installed)), logger)
	return client, plugin.(domain.ToolProvider)
}

func TestServiceToolsAreAnnotated(t *testing.T) {
	_, plugin := newPlugin(t)
	plugintest.RequireAnnotated(t, plugin)
}

func TestListServices(t *testing.T) {
	client, plugin := newPlugin(t, "postgres", "redis", "letsencrypt")
	client.Respond("postgres:list", "=====> Postgres services\nmain-db\nanalytics\n")
	client.Respond("redis:list", "NAME   VERSION      STATUS   EXPOSED PORTS  LINKS\ncache  redis:7.2  running  -              web\n")

	result := plugintest.Structured[ServiceList](t, plugintest.CallTool(t, plugin, "list_services", nil))
	if !slices.Equal(result.InstalledTypes, []string{"postgres", "redis"}) {
		t.Fatalf("installed types = %v", result.InstalledTypes)
	}
	if !slices.Equal(result.Services["postgres"], []string{"main-db", "analytics"}) {
		t.Fatalf("postgres services = %v", result.Services["postgres"])
	}
	if !slices.Equal(result.Services["redis"], []string{"cache"}) {
		t.Fatalf("redis services = %v", result.Services["redis"])
	}
}

func TestListServicesEmpty(t *testing.T) {
	client, plugin := newPlugin(t, "postgres")
	client.Respond("postgres:list", " !     There are no Postgres services\n")

	result := plugintest.Structured[ServiceList](t, plugintest.CallTool(t, plugin, "list_services", nil))
	if len(result.Services["postgres"]) != 0 {
		t.Fatalf("expected no services, got %v", result.Services["postgres"])
	}
}

func TestCreateServiceRequiresInstalledPlugin(t *testing.T) {
	client, plugin := newPlugin(t)

	result := plugintest.CallTool(t, plugin, "create_service", plugintest.Args{"service_type": "postgres", "name": "db"})
	plugintest.RequireError(t, result, "plugin:install https://github.com/dokku/dokku-postgres.git")
	if len(client.CallsTo("postgres:create")) != 0 {
		t.Fatal("must not run postgres:create when the plugin is missing")
	}
}

func TestCreateService(t *testing.T) {
	client, plugin := newPlugin(t, "postgres")

	result := plugintest.CallTool(t, plugin, "create_service", plugintest.Args{
		"service_type": "postgres", "name": "main-db", "image_version": "17.2",
	})
	plugintest.RequireSuccess(t, result)

	calls := client.CallsTo("postgres:create")
	if len(calls) != 1 || calls[0].String() != "postgres:create main-db --image-version 17.2" {
		t.Fatalf("unexpected calls: %v", calls)
	}
}

func TestCreateServiceRejectsInvalidName(t *testing.T) {
	client, plugin := newPlugin(t, "postgres")
	result := plugintest.CallTool(t, plugin, "create_service", plugintest.Args{"service_type": "postgres", "name": "Main DB"})
	plugintest.RequireError(t, result, "invalid service name")
	if len(client.Calls()) != 0 {
		t.Fatalf("expected no Dokku calls, got %v", client.Calls())
	}
}

func TestGetServiceInfoMasksCredentials(t *testing.T) {
	client, plugin := newPlugin(t, "postgres")
	client.Respond("postgres:info", `=====> main-db postgres service information
       Config dir:          /var/lib/dokku/services/postgres/main-db/data
       Dsn:                 postgres://postgres:s3cr3t-P4ss@dokku-postgres-main-db:5432/main_db
       Root password:       hunter2
       Status:              running
       Version:             postgres:17.2
`)
	client.Respond("postgres:links", "web\nworker\n")

	result := plugintest.CallTool(t, plugin, "get_service_info", plugintest.Args{"service_type": "postgres", "name": "main-db"})
	info := plugintest.Structured[ServiceInfo](t, result)

	if got := info.Info["Dsn"]; got != "postgres://postgres:***@dokku-postgres-main-db:5432/main_db" {
		t.Fatalf("Dsn = %q", got)
	}
	if got := info.Info["Root password"]; got != "***" {
		t.Fatalf("Root password = %q", got)
	}
	if info.Info["Status"] != "running" {
		t.Fatalf("Status = %q", info.Info["Status"])
	}
	if !slices.Equal(info.LinkedApps, []string{"web", "worker"}) {
		t.Fatalf("linked apps = %v", info.LinkedApps)
	}
	if text := plugintest.Text(result); strings.Contains(text, "s3cr3t") || strings.Contains(text, "hunter2") {
		t.Fatalf("result leaks credentials: %s", text)
	}
}

func TestLinkService(t *testing.T) {
	client, plugin := newPlugin(t, "redis")

	result := plugintest.CallTool(t, plugin, "link_service", plugintest.Args{
		"service_type": "redis", "name": "cache", "app_name": "web", "alias": "CACHE", "no_restart": true,
	})
	plugintest.RequireSuccess(t, result)

	calls := client.CallsTo("redis:link")
	if len(calls) != 1 || calls[0].String() != "redis:link cache web --alias CACHE --no-restart" {
		t.Fatalf("unexpected calls: %v", calls)
	}
}

func TestLinkServiceRejectsInvalidAlias(t *testing.T) {
	_, plugin := newPlugin(t, "redis")
	result := plugintest.CallTool(t, plugin, "link_service", plugintest.Args{
		"service_type": "redis", "name": "cache", "app_name": "web", "alias": "cache url",
	})
	plugintest.RequireError(t, result, "invalid alias")
}

func TestUnlinkService(t *testing.T) {
	client, plugin := newPlugin(t, "postgres")
	result := plugintest.CallTool(t, plugin, "unlink_service", plugintest.Args{
		"service_type": "postgres", "name": "db", "app_name": "web",
	})
	plugintest.RequireSuccess(t, result)
	if calls := client.CallsTo("postgres:unlink"); len(calls) != 1 || calls[0].String() != "postgres:unlink db web" {
		t.Fatalf("unexpected calls: %v", calls)
	}
}

func TestGetServiceLogsDoesNotFollow(t *testing.T) {
	client, plugin := newPlugin(t, "postgres")
	client.Respond("postgres:logs", "LOG:  database system is ready to accept connections\n")

	result := plugintest.CallTool(t, plugin, "get_service_logs", plugintest.Args{"service_type": "postgres", "name": "db"})
	plugintest.RequireSuccess(t, result)
	if calls := client.CallsTo("postgres:logs"); len(calls) != 1 || calls[0].String() != "postgres:logs db" {
		t.Fatalf("unexpected calls: %v", calls)
	}
}

func TestDestroyServiceRequiresConfirmation(t *testing.T) {
	client, plugin := newPlugin(t, "postgres")

	result := plugintest.CallTool(t, plugin, "destroy_service", plugintest.Args{
		"service_type": "postgres", "name": "db", "confirm_name": "other",
	})
	plugintest.RequireError(t, result, "confirm_name")
	if len(client.CallsTo("postgres:destroy")) != 0 {
		t.Fatal("must not destroy without confirmation")
	}

	result = plugintest.CallTool(t, plugin, "destroy_service", plugintest.Args{
		"service_type": "postgres", "name": "db", "confirm_name": "db",
	})
	plugintest.RequireSuccess(t, result)
	if calls := client.CallsTo("postgres:destroy"); len(calls) != 1 || calls[0].String() != "postgres:destroy db --force" {
		t.Fatalf("unexpected calls: %v", calls)
	}
}

func TestRedactCredentials(t *testing.T) {
	cases := map[string]string{
		"redis://:pass@host:6379":                    "redis://:***@host:6379",
		"mysql://mysql:p%40ss@dokku-mysql-db:3306/x": "mysql://mysql:***@dokku-mysql-db:3306/x",
		"postgres://host:5432/db":                    "postgres://host:5432/db",
		"no url here":                                "no url here",
	}
	for in, want := range cases {
		if got := RedactCredentials(in); got != want {
			t.Errorf("RedactCredentials(%q) = %q, want %q", in, got, want)
		}
	}
}

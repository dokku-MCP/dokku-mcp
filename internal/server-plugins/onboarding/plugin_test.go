package onboarding

import (
	"context"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"github.com/dokku-mcp/dokku-mcp/internal/dokku-api/dokkutest"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/app"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/app/infrastructure"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/core"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/deployment"
	deploymentDomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/deployment/domain"
	domainplugin "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/domain"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/letsencrypt"
	"github.com/dokku-mcp/dokku-mcp/internal/server-plugins/services"
	"github.com/dokku-mcp/dokku-mcp/pkg/config"
)

// allToolParams maps every tool of every plugin to its input parameters.
func allToolParams(t *testing.T) map[string]map[string]bool {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	client := dokkutest.NewFakeClient()
	providers := []domain.ToolProvider{
		app.NewAppsServerPlugin(infrastructure.NewDokkuApplicationRepository(client, logger), nil, logger, config.LogsConfig{}).(domain.ToolProvider),
		deployment.NewDeploymentServerPlugin(deploymentDomain.NewDeploymentTracker(), logger).(domain.ToolProvider),
		domainplugin.NewDomainServerPlugin(client, logger).(domain.ToolProvider),
		services.NewServicesServerPlugin(services.NewAdapter(client, nil), logger).(domain.ToolProvider),
		letsencrypt.NewLetsEncryptServerPlugin(client, logger).(domain.ToolProvider),
		core.NewCoreServerPlugin(client, logger, &config.ServerConfig{}).(domain.ToolProvider),
	}
	params := map[string]map[string]bool{}
	for _, provider := range providers {
		tools, err := provider.GetTools(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for _, tool := range tools {
			names := map[string]bool{}
			for name := range tool.Builder().InputSchema.Properties {
				names[name] = true
			}
			params[tool.Name] = names
		}
	}
	return params
}

func TestIntentMapReferencesRealTools(t *testing.T) {
	params := allToolParams(t)
	for intent, entry := range intentMap {
		toolParams, ok := params[entry.Tool]
		if !ok {
			t.Errorf("intent %q maps to unknown tool %q", intent, entry.Tool)
			continue
		}
		for _, p := range entry.Params {
			if !toolParams[p] {
				t.Errorf("intent %q lists parameter %q that %s does not accept", intent, p, entry.Tool)
			}
		}
	}
}

func TestQuickstartReferencesRealTools(t *testing.T) {
	params := allToolParams(t)
	identifier := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	for _, m := range regexp.MustCompile("`([^`]+)`").FindAllStringSubmatch(quickstartMarkdown, -1) {
		ref := m[1]
		// Resource URIs and the app_doctor prompt are not tools.
		if strings.Contains(ref, "://") || ref == "app_doctor" {
			continue
		}
		if !identifier.MatchString(ref) {
			t.Errorf("quickstart has an unexpected code span %q; only tool names, resource URIs and prompts are checked", ref)
			continue
		}
		if _, ok := params[ref]; !ok {
			t.Errorf("quickstart mentions unknown tool %q", ref)
		}
	}
}

package onboarding

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	mcpserver "github.com/dokku-mcp/dokku-mcp/internal/server"
	serverDomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugin/domain"
	onbDomain "github.com/dokku-mcp/dokku-mcp/internal/server-plugins/onboarding/domain"
	"github.com/mark3labs/mcp-go/mcp"
)

// OnboardingServerPlugin provides discovery and onboarding resources
type OnboardingServerPlugin struct {
	provider mcpserver.ServerPluginProvider
}

func NewOnboardingServerPlugin() *OnboardingServerPlugin {
	return &OnboardingServerPlugin{}
}

// SetProvider allows late injection to avoid Fx cycles
func (p *OnboardingServerPlugin) SetProvider(provider mcpserver.ServerPluginProvider) {
	p.provider = provider
}

// ServerPlugin interface
func (p *OnboardingServerPlugin) ID() string   { return "onboarding" }
func (p *OnboardingServerPlugin) Name() string { return "Onboarding & Discovery" }
func (p *OnboardingServerPlugin) Description() string {
	return "LLM onboarding resources and prompt discovery"
}
func (p *OnboardingServerPlugin) Version() string { return "0.3.0" }
func (p *OnboardingServerPlugin) DokkuPluginName() string {
	return "" // always active
}

// ResourceProvider implementation
func (p *OnboardingServerPlugin) GetResources(ctx context.Context) ([]serverDomain.Resource, error) {
	return []serverDomain.Resource{
		{
			URI:         "dokku://onboarding/quickstart",
			Name:        "Quickstart",
			Description: "Start here: overview of tools, prompts, resources, planner, and safe usage",
			MIMEType:    "text/markdown",
			Handler:     p.handleQuickstartResource,
		},
		{
			URI:         "dokku://onboarding/capabilities",
			Name:        "Capabilities Index",
			Description: "Index of tools, resources, prompts, with examples and safety notes",
			MIMEType:    "application/json",
			Handler:     p.handleCapabilitiesIndexResource,
		},
		{
			URI:         "dokku://onboarding/intent-map",
			Name:        "Intent Map",
			Description: "Mapping of generic platform intents and synonyms to Dokku tools",
			MIMEType:    "application/json",
			Handler:     p.handleIntentMapResource,
		},
	}, nil
}

// quickstartMarkdown orients a model on first use. Keep tool and parameter
// names in sync with the plugins; onboarding tests check them.
const quickstartMarkdown = `# Quickstart

This MCP server manages a Dokku host. Tools appear only when the Dokku plugin they need is installed
(for example the Let's Encrypt tools need dokku-letsencrypt).

## Core flow
1. ` + "`create_app`" + ` → { "name": "my-app" }
2. ` + "`deploy_app`" + ` → { "app_name": "my-app", "repo_url": "https://github.com/acme/app.git", "git_ref": "main" }
   Returns a deployment_id immediately; the build continues in the background.
3. ` + "`get_deployment_status`" + ` → { "deployment_id": "..." } until "done" is true. Failed builds include the build log tail.
4. ` + "`scale_app`" + ` → { "app_name": "my-app", "process_type": "web", "instances": 2 }
5. ` + "`get_app_status`" + ` → { "app_name": "my-app" }

## Configuration and data
- Environment variables: ` + "`configure_app`" + ` → { "app_name": "my-app", "config": { "KEY": "value" }, "restart": true }
- Datastores: ` + "`create_service`" + ` then ` + "`link_service`" + ` (sets DATABASE_URL, REDIS_URL, ...); ` + "`list_services`" + ` shows what is installed.

## Domains and HTTPS
- ` + "`add_app_domain`" + `, ` + "`get_app_domains`" + `, then ` + "`enable_letsencrypt`" + ` once DNS points at the server.

## Operating and troubleshooting
- Logs: ` + "`get_runtime_logs`" + ` (recent), ` + "`follow_runtime_logs`" + ` (live, a few seconds), ` + "`get_failed_deploy_logs`" + ` (crashed releases)
- Lifecycle: ` + "`restart_app`" + `, ` + "`stop_app`" + `, ` + "`start_app`" + `
- Rollback: ` + "`rollback_app`" + ` → { "app_name": "my-app", "git_ref": "<known-good ref>" } redeploys from the last repository
- Prompt ` + "`app_doctor`" + ` walks through a diagnosis

## Safety
- Read before you write: check ` + "`get_app_status`" + ` or ` + "`get_app_domains`" + ` first.
- Tools carry readOnlyHint / destructiveHint annotations; destructive tools (destroy_service, remove_app_domain,
  stop_app, ...) should only run when the user asked for them.
- The server may block commands through its allowlist/blacklist. Report such errors instead of retrying.

## Discover
- All tools, resources and prompts: ` + "`dokku://onboarding/capabilities`" + `
- Goal → tool mapping: ` + "`dokku://onboarding/intent-map`" + `
- Server info and plugins: ` + "`dokku://core/server/info`" + `, ` + "`dokku://core/plugins`" + `
`

// Handlers
func (p *OnboardingServerPlugin) handleQuickstartResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	md := quickstartMarkdown
	return []mcp.ResourceContents{mcp.TextResourceContents{URI: req.Params.URI, MIMEType: "text/markdown", Text: md}}, nil
}

func (p *OnboardingServerPlugin) handleCapabilitiesIndexResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Build a minimal index from the provider
	index := onbDomain.NewCapabilityIndex()
	// Tools
	tools := make([]onbDomain.CapabilityTool, 0)
	for _, tp := range p.provider.GetToolProviders() {
		ts, err := tp.GetTools(ctx)
		if err == nil {
			for _, t := range ts {
				ex := make([]onbDomain.CapabilityToolExample, 0)
				if t.Name == "deploy_app" {
					ex = append(ex, onbDomain.CapabilityToolExample{
						Tool: t.Name,
						Params: onbDomain.CapabilityToolExampleParams{
							AppName: "my-app",
							RepoURL: "https://github.com/acme/app.git",
							GitRef:  "main",
						},
					})
				}
				tools = append(tools, onbDomain.CapabilityTool{Name: t.Name, Description: t.Description, Examples: ex})
			}
		}
	}
	// Resources
	resources := make([]onbDomain.CapabilityResource, 0)
	for _, rp := range p.provider.GetResourceProviders() {
		rs, err := rp.GetResources(ctx)
		if err == nil {
			for _, r := range rs {
				resources = append(resources, onbDomain.CapabilityResource{URI: r.URI, Name: r.Name, Description: r.Description, MIMEType: r.MIMEType})
			}
		}
	}
	// Prompts
	caps, _ := p.aggregatePrompts(ctx)

	index.Tools = tools
	index.Resources = resources
	index.Prompts = caps.Prompts

	b, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal capabilities index: %w", err)
	}
	return []mcp.ResourceContents{mcp.TextResourceContents{URI: req.Params.URI, MIMEType: "application/json", Text: string(b)}}, nil
}

func (p *OnboardingServerPlugin) handleIntentMapResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	jsonData, err := json.MarshalIndent(intentMap, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal intent map: %w", err)
	}
	return []mcp.ResourceContents{mcp.TextResourceContents{URI: req.Params.URI, MIMEType: "application/json", Text: string(jsonData)}}, nil
}

// intentMap maps generic platform intents to tools and their parameters.
var intentMap = map[string]onbDomain.IntentEntry{
	"create":          {Synonyms: []string{"new app", "provision", "bootstrap"}, Tool: "create_app", Params: []string{"name"}},
	"deploy":          {Synonyms: []string{"release", "ship", "publish", "roll out", "push code"}, Tool: "deploy_app", Params: []string{"app_name", "repo_url", "git_ref"}},
	"deploy_status":   {Synonyms: []string{"is it deployed", "build status", "build logs"}, Tool: "get_deployment_status", Params: []string{"deployment_id"}},
	"rollback":        {Synonyms: []string{"revert", "undo deploy", "previous version"}, Tool: "rollback_app", Params: []string{"app_name", "git_ref"}},
	"scale":           {Synonyms: []string{"autoscale", "increase instances", "add nodes", "replicas"}, Tool: "scale_app", Params: []string{"app_name", "process_type", "instances"}},
	"restart":         {Synonyms: []string{"reboot", "bounce", "reload"}, Tool: "restart_app", Params: []string{"app_name"}},
	"status":          {Synonyms: []string{"health", "state", "check app"}, Tool: "get_app_status", Params: []string{"app_name"}},
	"configure":       {Synonyms: []string{"set env", "set variables", "secrets", "config"}, Tool: "configure_app", Params: []string{"app_name", "config"}},
	"logs":            {Synonyms: []string{"output", "errors", "what happened"}, Tool: "get_runtime_logs", Params: []string{"app_name", "lines"}},
	"tail_logs":       {Synonyms: []string{"watch logs", "live logs", "follow"}, Tool: "follow_runtime_logs", Params: []string{"app_name", "seconds"}},
	"crash":           {Synonyms: []string{"failed deploy", "boot failure", "healthcheck failed"}, Tool: "get_failed_deploy_logs", Params: []string{"app_name"}},
	"database":        {Synonyms: []string{"postgres", "mysql", "redis", "datastore", "cache"}, Tool: "create_service", Params: []string{"service_type", "name"}},
	"attach_database": {Synonyms: []string{"connect database", "DATABASE_URL", "link db"}, Tool: "link_service", Params: []string{"service_type", "name", "app_name"}},
	"domain":          {Synonyms: []string{"custom domain", "hostname", "vhost"}, Tool: "add_app_domain", Params: []string{"app_name", "domain_name"}},
	"https":           {Synonyms: []string{"ssl", "tls", "certificate", "letsencrypt"}, Tool: "enable_letsencrypt", Params: []string{"app_name", "email"}},
	"access":          {Synonyms: []string{"ssh key", "give access", "deploy key"}, Tool: "add_ssh_key", Params: []string{"name", "public_key"}},
	"registry":        {Synonyms: []string{"docker login", "ghcr", "private images"}, Tool: "registry_login", Params: []string{"server", "username", "password"}},
}

// aggregatePrompts collects prompts across active plugins
func (p *OnboardingServerPlugin) aggregatePrompts(ctx context.Context) (onbDomain.PromptsCapabilities, error) {
	prompts := make([]onbDomain.PromptMeta, 0)
	for _, pp := range p.provider.GetPromptProviders() {
		ps, err := pp.GetPrompts(ctx)
		if err == nil {
			for _, pr := range ps {
				prompts = append(prompts, onbDomain.PromptMeta{Plugin: pp.ID(), Name: pr.Name, Description: pr.Description})
			}
		}
	}
	return onbDomain.PromptsCapabilities{Version: "0.1.0", GeneratedAt: time.Now().UTC(), Prompts: prompts}, nil
}

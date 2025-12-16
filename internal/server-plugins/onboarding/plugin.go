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

// Handlers
func (p *OnboardingServerPlugin) handleQuickstartResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	md := "# Quickstart\n\n" +
		"Welcome. This MCP server exposes tools and resources to manage Dokku server.\n\n" +
		"## What you can do\n" +
		"- Create apps: `create_app`\n" +
		"- Deploy from Git: `deploy_app`\n" +
		"- Scale processes: `scale_app`\n" +
		"- Configure env: `configure_app`\n" +
		"- Check status: `get_app_status`\n\n" +
		"## Core flow (copy‑paste ready)\n" +
		"1) `create_app` → `{ name: \"my-app\" }`\n" +
		"2) `deploy_app` → `{ app_name: \"my-app\", repo_url: \"https://github.com/acme/app.git\", git_ref: \"main\" }`\n" +
		"3) `scale_app` → `{ app_name: \"my-app\", process_type: \"web\", instances: 2 }`\n" +
		"4) `get_app_status` → `{ app_name: \"my-app\" }`\n\n" +
		"## Discover & learn\n" +
		"- Generic goals → tools: `dokku://onboarding/intent-map`\n" +
		"- Recipes (blue/green, etc.): `dokku://onboarding/examples`\n" +
		"- Prompts (e.g., `app_doctor`): `dokku://onboarding/capabilities`\n" +
		"- Server info (status, plugins): `dokku://core/server/info`, `dokku://core/plugins`\n\n" +
		"## Planner (optional)\n" +
		"Use `suggest_tools` to translate a natural goal into a safe plan.\n" +
		"Example goal: \"deploy latest and scale to 2\" → returns read‑first checks, then mutating steps with `confirm: true`.\n\n" +
		"## Safety & best practices\n" +
		"- Do read‑only checks (`get_app_status`) before mutations\n" +
		"- Use confirmations on deploy/scale/config (`confirm: true`)\n" +
		"- Keep a rollback plan (redeploy previous `git_ref`)\n\n" +
		"## Troubleshooting\n" +
		"- App missing after deploy → ensure `create_app` used same name as `app_name`\n" +
		"- Bad deploy → redeploy known‑good `git_ref`, then `get_app_status`\n" +
		"- Need help → use prompt `app_doctor` with your `app_name`\n"
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
							AppName:      "my-app",
							RepoURL:      "https://github.com/acme/app.git",
							GitRef:       "main",
							ValidateOnly: true,
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
	mapping := map[string]onbDomain.IntentEntry{
		"deploy":    {Synonyms: []string{"release", "ship", "publish", "roll out", "push code"}, Tool: "deploy_app", Params: []string{"app_name", "repo_url", "git_ref"}},
		"scale":     {Synonyms: []string{"autoscale", "increase instances", "add nodes", "replicas"}, Tool: "scale_app", Params: []string{"app_name", "process_type", "instances"}},
		"status":    {Synonyms: []string{"health", "state", "check app", "diagnose"}, Tool: "get_app_status", Params: []string{"app_name"}},
		"configure": {Synonyms: []string{"set env", "set variables", "secrets", "config"}, Tool: "configure_app", Params: []string{"app_name", "config"}},
		"create":    {Synonyms: []string{"new app", "provision", "bootstrap"}, Tool: "create_app", Params: []string{"name", "buildpack"}},
	}
	jsonData, err := json.MarshalIndent(mapping, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal intent map: %w", err)
	}
	return []mcp.ResourceContents{mcp.TextResourceContents{URI: req.Params.URI, MIMEType: "application/json", Text: string(jsonData)}}, nil
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

package prompts

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type promptConfig struct {
	name        string
	description string
	template    string
}

type PromptsManager struct {
	logger *slog.Logger
}

func NewPromptsManager(logger *slog.Logger) *PromptsManager {
	return &PromptsManager{
		logger: logger,
	}
}

func (pm *PromptsManager) RegisterPrompts(mcpServer *server.MCPServer) error {
	pm.logger.Debug("Registering prompts")

	prompts := []promptConfig{
		{
			name:        "diagnose_application",
			description: "Analyser les problèmes potentiels d'une application Dokku",
			template: `Veuillez diagnostiquer l'application Dokku "%s". 
				
Analysez les aspects suivants :
1. État de l'application et des processus
2. Configuration des variables d'environnement
3. Problèmes potentiels de déploiement
5. Vérifications de sécurité

Utilisez les outils disponibles pour récupérer les informations nécessaires et fournir un rapport détaillé.`,
		},
		{
			name:        "optimize_application",
			description: "Générer des recommandations d'optimisation pour une application Dokku",
			template: `Veuillez analyser l'application Dokku "%s" et fournir des recommandations d'optimisation.
				
Domaines d'analyse :
1. Performance des processus et scaling
2. Configuration des ressources (CPU/mémoire)
3. Optimisation des variables d'environnement
4. Configuration des domaines et SSL
5. Optimisations de buildpack
6. Stratégies de cache
7. Surveillance et observabilité

Fournissez des recommandations spécifiques et actionnables.`,
		},
	}

	for _, promptCfg := range prompts {
		if err := pm.registerPrompt(mcpServer, promptCfg); err != nil {
			return fmt.Errorf("échec de l'enregistrement du prompt %s: %w", promptCfg.name, err)
		}
	}

	pm.logger.Debug("Prompts registered successfully")
	return nil
}

func (pm *PromptsManager) registerPrompt(mcpServer *server.MCPServer, config promptConfig) error {
	prompt := mcp.NewPrompt(
		config.name,
		mcp.WithPromptDescription(config.description),
		mcp.WithArgument("app_name",
			mcp.ArgumentDescription("Name of the application to diagnose"),
			mcp.RequiredArgument(),
		),
	)

	handler := pm.createPromptHandler(config)
	mcpServer.AddPrompt(prompt, handler)

	return nil
}

func (pm *PromptsManager) createPromptHandler(config promptConfig) func(context.Context, mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		appName := pm.extractAppNameFromRequest(req)

		message := mcp.NewPromptMessage(
			mcp.RoleUser,
			mcp.NewTextContent(fmt.Sprintf(config.template, appName)),
		)

		return mcp.NewGetPromptResult(
			fmt.Sprintf("Dokku application diagnosis: %s", config.name),
			[]mcp.PromptMessage{message},
		), nil
	}
}

func (pm *PromptsManager) extractAppNameFromRequest(req mcp.GetPromptRequest) string {
	if req.Params.Arguments == nil {
		return ""
	}

	if name, ok := req.Params.Arguments["app_name"]; ok {
		return name
	}

	return ""
}

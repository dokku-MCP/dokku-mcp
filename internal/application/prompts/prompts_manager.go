package prompts

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	domain "github.com/alex-galey/dokku-mcp/internal/domain"
)

// PromptsManager gère l'orchestration des prompts MCP pour Dokku
// Il utilise les templates métier du domaine et gère leur enregistrement MCP
type PromptsManager struct {
	logger    *slog.Logger
	templates *domain.ApplicationPromptTemplates
}

// NewPromptsManager crée un nouveau gestionnaire de prompts
func NewPromptsManager(logger *slog.Logger) *PromptsManager {
	return &PromptsManager{
		logger:    logger,
		templates: domain.NewApplicationPromptTemplates(),
	}
}

// RegisterPrompts enregistre les prompts MCP en utilisant les templates du domaine
func (pm *PromptsManager) RegisterPrompts(mcpServer *server.MCPServer) error {
	pm.logger.Debug("Enregistrement des prompts MCP à partir des templates métier")

	// Récupération des templates métier depuis le domaine
	domainTemplates := pm.templates.GetAllPromptTemplates()

	// Enregistrement de chaque template
	for _, template := range domainTemplates {
		if err := pm.registerPrompt(mcpServer, template); err != nil {
			return fmt.Errorf("échec de l'enregistrement du prompt %s: %w", template.Name, err)
		}
		pm.logger.Debug("Prompt enregistré", "name", template.Name)
	}

	pm.logger.Info("Tous les prompts MCP enregistrés avec succès", "count", len(domainTemplates))
	return nil
}

// registerPrompt enregistre un template de prompt du domaine dans MCP
func (pm *PromptsManager) registerPrompt(mcpServer *server.MCPServer, template domain.PromptTemplate) error {
	// Création du prompt MCP avec les métadonnées du domaine
	prompt := mcp.NewPrompt(
		template.Name,
		mcp.WithPromptDescription(template.Description),
		// Ajout du paramètre app_name requis pour tous les prompts
		mcp.WithArgument("app_name",
			mcp.ArgumentDescription("Nom de l'application Dokku à analyser"),
			mcp.RequiredArgument(),
		),
	)

	// Création du handler qui utilise le template métier
	handler := pm.createPromptHandler(template)
	mcpServer.AddPrompt(prompt, handler)

	return nil
}

// createPromptHandler crée un handler MCP pour un template métier
func (pm *PromptsManager) createPromptHandler(template domain.PromptTemplate) func(context.Context, mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return func(ctx context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		pm.logger.Debug("Génération de prompt à partir du template métier", "template", template.Name)

		// Extraction des paramètres
		appName := pm.extractAppNameFromRequest(req)
		if appName == "" {
			pm.logger.Warn("Nom d'application manquant dans la requête de prompt", "template", template.Name)
			return nil, fmt.Errorf("le paramètre app_name est requis pour le prompt %s", template.Name)
		}

		// Utilisation du template métier du domaine
		promptText := fmt.Sprintf(template.Template, appName)

		// Création du message MCP
		message := mcp.NewPromptMessage(
			mcp.RoleUser,
			mcp.NewTextContent(promptText),
		)

		// Retour du résultat avec titre descriptif
		title := fmt.Sprintf("Analyse Dokku: %s pour '%s'", template.Name, appName)
		return mcp.NewGetPromptResult(title, []mcp.PromptMessage{message}), nil
	}
}

// extractAppNameFromRequest extrait le nom de l'application depuis la requête MCP
func (pm *PromptsManager) extractAppNameFromRequest(req mcp.GetPromptRequest) string {
	if req.Params.Arguments == nil {
		return ""
	}

	if name, ok := req.Params.Arguments["app_name"]; ok {
		return name
	}

	return ""
}

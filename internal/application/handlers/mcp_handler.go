package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/alex-galey/dokku-mcp/internal/application/dto"
	"github.com/alex-galey/dokku-mcp/internal/application/usecases"
	"github.com/alex-galey/dokku-mcp/internal/domain"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPApplicationHandler implements domain.MCPHandler for application operations
type MCPApplicationHandler struct {
	applicationUseCase *usecases.ApplicationUseCase
	applicationMapper  *dto.ApplicationMapper
	logger             *slog.Logger
}

// NewMCPApplicationHandler creates a new MCP application handler
func NewMCPApplicationHandler(
	applicationUseCase *usecases.ApplicationUseCase,
	applicationMapper *dto.ApplicationMapper,
	logger *slog.Logger,
) domain.MCPHandler {
	return &MCPApplicationHandler{
		applicationUseCase: applicationUseCase,
		applicationMapper:  applicationMapper,
		logger:             logger,
	}
}

// HandleApplicationsResource handles the applications resource request
func (h *MCPApplicationHandler) HandleApplicationsResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	h.logger.Debug("Processing applications resource request")

	// Use the use case to retrieve applications
	apps, err := h.applicationUseCase.GetAllApplications(ctx)
	if err != nil {
		h.logger.Error("Failed to retrieve applications", "error", err)
		return nil, fmt.Errorf("failed to retrieve applications: %w", err)
	}

	// Convert to DTOs
	appDTOs := h.applicationMapper.ToDTOs(apps)

	// Create individual resource for each application
	resources := make([]mcp.ResourceContents, len(appDTOs))
	for i, appDTO := range appDTOs {
		// Serialize each application individually
		jsonData, err := json.MarshalIndent(appDTO, "", "  ")
		if err != nil {
			h.logger.Error("Failed to serialize JSON for application",
				"app_name", appDTO.Name,
				"error", err)
			continue
		}

		// Create individual resource with unique URI
		resources[i] = mcp.TextResourceContents{
			URI:      fmt.Sprintf("dokku://app/%s", appDTO.Name),
			MIMEType: "application/json",
			Text:     string(jsonData),
		}
	}

	h.logger.Debug("Application resources created successfully", "count", len(resources))
	return resources, nil
}

// HandleCreateApplication handles the create application tool request
func (h *MCPApplicationHandler) HandleCreateApplication(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	h.logger.Debug("Processing create application request")

	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("Le paramètre 'name' est requis et doit être une chaîne"), nil
	}

	// Use the use case to create the application
	cmd := usecases.CreateApplicationCommand{
		Name: name,
	}

	if err := h.applicationUseCase.CreateApplication(ctx, cmd); err != nil {
		h.logger.Error("Failed to create application", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Échec de création de l'application: %v", err)), nil
	}

	h.logger.Info("Application created successfully", "app_name", name)
	return mcp.NewToolResultText(fmt.Sprintf("✅ Application '%s' créée avec succès", name)), nil
}

// HandleDeployApplication handles the deploy application tool request
func (h *MCPApplicationHandler) HandleDeployApplication(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	h.logger.Debug("Processing deploy application request")

	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("Le paramètre 'name' est requis"), nil
	}

	gitRef := req.GetString("git_ref", "main")

	// Use the use case to deploy the application
	cmd := usecases.DeployApplicationCommand{
		Name:   name,
		GitRef: gitRef,
	}

	if err := h.applicationUseCase.DeployApplication(ctx, cmd); err != nil {
		h.logger.Error("Failed to deploy application", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Échec du déploiement: %v", err)), nil
	}

	h.logger.Info("Deployment completed successfully",
		"app_name", name,
		"git_ref", gitRef)
	return mcp.NewToolResultText(fmt.Sprintf("✅ Déploiement de '%s' réussi (ref: %s)", name, gitRef)), nil
}

// HandleScaleApplication handles the scale application tool request
func (h *MCPApplicationHandler) HandleScaleApplication(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	h.logger.Debug("Processing scale application request")

	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("Le paramètre 'name' est requis"), nil
	}

	processType := req.GetString("process_type", "web")
	scale, err := req.RequireFloat("scale")
	if err != nil {
		return mcp.NewToolResultError("Le paramètre 'scale' est requis et doit être un nombre"), nil
	}

	// Use the use case to scale the application
	cmd := usecases.ScaleApplicationCommand{
		Name:        name,
		ProcessType: processType,
		Scale:       int(scale),
	}

	if err := h.applicationUseCase.ScaleApplication(ctx, cmd); err != nil {
		h.logger.Error("Failed to scale application", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Échec du scaling: %v", err)), nil
	}

	h.logger.Info("Scaling completed successfully",
		"app_name", name,
		"process_type", processType,
		"scale", int(scale))
	return mcp.NewToolResultText(fmt.Sprintf("✅ Scaling de '%s' réussi (%s: %d instances)", name, processType, int(scale))), nil
}

// HandleSetApplicationConfig handles the set application config tool request
func (h *MCPApplicationHandler) HandleSetApplicationConfig(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	h.logger.Debug("Processing set application config request")

	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError("Le paramètre 'name' est requis"), nil
	}

	// Extract config parameter using GetArguments()
	args := req.GetArguments()
	configParam, ok := args["config"]
	if !ok {
		return mcp.NewToolResultError("Le paramètre 'config' est requis"), nil
	}

	configMap, ok := configParam.(map[string]interface{})
	if !ok {
		return mcp.NewToolResultError("Le paramètre 'config' doit être un objet"), nil
	}

	// Convert to string map
	config := make(map[string]string)
	for key, value := range configMap {
		if strValue, ok := value.(string); ok {
			config[key] = strValue
		} else {
			return mcp.NewToolResultError(fmt.Sprintf("La valeur de '%s' doit être une chaîne", key)), nil
		}
	}

	// Use the use case to set application config
	cmd := usecases.SetConfigCommand{
		Name:   name,
		Config: config,
	}

	if err := h.applicationUseCase.SetApplicationConfig(ctx, cmd); err != nil {
		h.logger.Error("Failed to set application config", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Échec de configuration: %v", err)), nil
	}

	h.logger.Info("Application configuration set successfully",
		"app_name", name,
		"config_keys", len(config))
	return mcp.NewToolResultText(fmt.Sprintf("✅ Configuration de '%s' mise à jour (%d variables)", name, len(config))), nil
}

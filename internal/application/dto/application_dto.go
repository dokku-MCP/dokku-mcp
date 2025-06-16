package dto

import (
	"time"

	app_domain "github.com/alex-galey/dokku-mcp/internal/domain/dokku/application"
)

// ApplicationDTO représente les données d'une application pour la couche application
type ApplicationDTO struct {
	Name       string             `json:"name"`
	State      string             `json:"state"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
	Config     *ConfigurationDTO  `json:"configuration,omitempty"`
	Deployment *DeploymentInfoDTO `json:"deployment_info,omitempty"`
}

// ConfigurationDTO représente la configuration d'une application
type ConfigurationDTO struct {
	Buildpack       string            `json:"buildpack,omitempty"`
	Domains         []string          `json:"domains"`
	EnvironmentVars map[string]string `json:"environment_variables"`
	Processes       []ProcessDTO      `json:"processes"`
}

// ProcessDTO représente un processus d'application
type ProcessDTO struct {
	Type    string `json:"type"`
	Command string `json:"command"`
	Scale   int    `json:"scale"`
}

// DeploymentInfoDTO représente les informations de déploiement
type DeploymentInfoDTO struct {
	CurrentGitRef   string     `json:"current_git_ref,omitempty"`
	LastDeployedAt  *time.Time `json:"last_deployed_at,omitempty"`
	BuildImage      string     `json:"build_image,omitempty"`
	RunImage        string     `json:"run_image,omitempty"`
	DeploymentCount int        `json:"deployment_count"`
}

// ApplicationMapper convertit entre le domaine et les DTOs
type ApplicationMapper struct{}

// NewApplicationMapper crée un nouveau mapper d'application
func NewApplicationMapper() *ApplicationMapper {
	return &ApplicationMapper{}
}

// ToDTO convertit une entité Application en DTO
func (m *ApplicationMapper) ToDTO(app *app_domain.Application) *ApplicationDTO {
	if app == nil {
		return nil
	}

	dto := &ApplicationDTO{
		Name:      app.Name().Value(),
		State:     string(app.State().Value()),
		CreatedAt: app.CreatedAt(),
		UpdatedAt: app.UpdatedAt(),
	}

	// Mapper la configuration si elle existe
	if config := app.Configuration(); config != nil {
		dto.Config = m.mapConfiguration(config)
	}

	return dto
}

// ToDTOs convertit une liste d'entités Application en DTOs
func (m *ApplicationMapper) ToDTOs(apps []*app_domain.Application) []*ApplicationDTO {
	if apps == nil {
		return nil
	}

	dtos := make([]*ApplicationDTO, len(apps))
	for i, app := range apps {
		dtos[i] = m.ToDTO(app)
	}

	return dtos
}

// mapConfiguration convertit la configuration du domaine en DTO
func (m *ApplicationMapper) mapConfiguration(config *app_domain.ApplicationConfiguration) *ConfigurationDTO {
	configDTO := &ConfigurationDTO{
		Domains:         m.mapDomains(config),
		EnvironmentVars: m.mapEnvironmentVars(config),
		Processes:       m.mapProcesses(config),
	}

	// Mapper le buildpack si présent
	if buildpack := m.getBuildpack(config); buildpack != "" {
		configDTO.Buildpack = buildpack
	}

	return configDTO
}

// mapDomains convertit les domaines en slice de strings
func (m *ApplicationMapper) mapDomains(config *app_domain.ApplicationConfiguration) []string {
	// Note: Cette méthode devra être adaptée selon l'interface réelle de ApplicationConfiguration
	// Pour l'instant, on retourne une slice vide
	return []string{}
}

// mapEnvironmentVars convertit les variables d'environnement
func (m *ApplicationMapper) mapEnvironmentVars(config *app_domain.ApplicationConfiguration) map[string]string {
	// Note: Cette méthode devra être adaptée selon l'interface réelle de ApplicationConfiguration
	// Pour l'instant, on retourne une map vide
	return map[string]string{}
}

// mapProcesses convertit les processus en DTOs
func (m *ApplicationMapper) mapProcesses(config *app_domain.ApplicationConfiguration) []ProcessDTO {
	// Note: Cette méthode devra être adaptée selon l'interface réelle de ApplicationConfiguration
	// Pour l'instant, on retourne une slice vide
	return []ProcessDTO{}
}

// getBuildpack récupère le nom du buildpack
func (m *ApplicationMapper) getBuildpack(config *app_domain.ApplicationConfiguration) string {
	// Note: Cette méthode devra être adaptée selon l'interface réelle de ApplicationConfiguration
	// Pour l'instant, on retourne une chaîne vide
	return ""
}

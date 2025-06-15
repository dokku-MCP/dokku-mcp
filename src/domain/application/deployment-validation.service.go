package application

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// DeploymentValidationService service domain pour valider les déploiements
type DeploymentValidationService struct {
	validationService *ValidationService
}

// NewDeploymentValidationService crée un nouveau service de validation de déploiement
func NewDeploymentValidationService() *DeploymentValidationService {
	return &DeploymentValidationService{
		validationService: NewValidationService(),
	}
}

// DeploymentContext contexte d'un déploiement
type DeploymentContext struct {
	Application    *Application
	GitRef         *GitRef
	Buildpack      *BuildpackName
	ForceClean     bool
	NoCache        bool
	Environment    string // production, staging, development
	PreviousDeploy *Deployment
}

// PreDeploymentCheck effectue les vérifications avant déploiement
func (s *DeploymentValidationService) PreDeploymentCheck(ctx context.Context, deployCtx *DeploymentContext) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// Validation de base de l'application
	appResult := s.validationService.ValidateApplication(ctx, deployCtx.Application)
	s.mergeResults(result, appResult)

	// Vérifications spécifiques au déploiement
	s.validateDeploymentReadiness(deployCtx, result)
	s.validateGitReference(deployCtx.GitRef, result)
	s.validateBuildpackCompatibility(deployCtx, result)
	s.validateEnvironmentConsistency(deployCtx, result)
	s.validateResourceAvailability(deployCtx, result)

	return result
}

// PostDeploymentValidation effectue les vérifications après déploiement
func (s *DeploymentValidationService) PostDeploymentValidation(ctx context.Context, deployment *Deployment, app *Application) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// Vérifier l'état du déploiement
	if deployment.Status != DeploymentStatusSucceeded {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "deployment.status",
			Message: fmt.Sprintf("Le déploiement a échoué: %s", deployment.ErrorMsg),
			Code:    "DEPLOYMENT_FAILED",
		})
		return result
	}

	// Vérifications de cohérence post-déploiement
	s.validateDeploymentConsistency(deployment, app, result)
	s.validateApplicationHealth(app, result)

	return result
}

// ValidateRollback valide une opération de rollback
func (s *DeploymentValidationService) ValidateRollback(ctx context.Context, app *Application, targetVersion string, currentDeployment *Deployment) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// Vérifier que l'application permet le rollback
	if !app.IsDeployed() {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "application.state",
			Message: "L'application doit être déployée pour effectuer un rollback",
			Code:    "APP_NOT_DEPLOYED",
		})
		return result
	}

	// Vérifier la version cible
	if targetVersion == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "target_version",
			Message: "La version cible du rollback est requise",
			Code:    "TARGET_VERSION_REQUIRED",
		})
	}

	// Avertissement si rollback vers une version très ancienne
	if currentDeployment != nil && s.isOldDeployment(currentDeployment) {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "target_version",
			Message: "Rollback vers une version ancienne détecté, vérifiez la compatibilité",
			Code:    "OLD_VERSION_ROLLBACK",
		})
	}

	return result
}

// validateDeploymentReadiness vérifie si l'application est prête pour le déploiement
func (s *DeploymentValidationService) validateDeploymentReadiness(deployCtx *DeploymentContext, result *ValidationResult) {
	app := deployCtx.Application

	// L'application ne doit pas être en cours de déploiement
	if app.State() == StateDeployed && deployCtx.PreviousDeploy != nil && deployCtx.PreviousDeploy.IsRunning() {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "application.state",
			Message: "Un déploiement est déjà en cours",
			Code:    "DEPLOYMENT_IN_PROGRESS",
		})
	}

	// Vérifier l'état de l'application
	if app.State() == StateError && !deployCtx.ForceClean {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "application.state",
			Message: "L'application est en erreur, utilisez force_clean si nécessaire",
			Code:    "APP_ERROR_STATE",
		})
	}

	// Vérifier la configuration minimale
	if len(app.Config().Processes) == 0 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "application.processes",
			Message: "Aucun processus configuré, le déploiement pourrait échouer",
			Code:    "NO_PROCESSES_CONFIGURED",
		})
	}
}

// validateGitReference valide la référence Git pour le déploiement
func (s *DeploymentValidationService) validateGitReference(gitRef *GitRef, result *ValidationResult) {
	if gitRef == nil {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "git_ref",
			Message: "La référence Git est requise pour le déploiement",
			Code:    "GIT_REF_REQUIRED",
		})
		return
	}

	// Validation spécifique selon le type de référence
	if gitRef.IsSHA() {
		if len(gitRef.Value()) < 7 {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "git_ref",
				Message: "SHA Git très court, risque d'ambiguïté",
				Code:    "SHORT_SHA",
			})
		}
	} else if gitRef.IsBranch() {
		// Avertissement pour les branches de développement en production
		branchName := strings.ToLower(gitRef.Value())
		if strings.Contains(branchName, "dev") || strings.Contains(branchName, "test") {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "git_ref",
				Message: "Branche de développement détectée, vérifiez l'environnement cible",
				Code:    "DEV_BRANCH_WARNING",
			})
		}
	}
}

// validateBuildpackCompatibility valide la compatibilité du buildpack
func (s *DeploymentValidationService) validateBuildpackCompatibility(deployCtx *DeploymentContext, result *ValidationResult) {
	if deployCtx.Buildpack == nil {
		return // Buildpack optionnel
	}

	buildpack := deployCtx.Buildpack
	app := deployCtx.Application

	// Vérification de sécurité pour les buildpacks custom
	if !buildpack.IsOfficial() && !buildpack.IsURL() {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "buildpack",
			Message: "Buildpack non-officiel détecté, vérifiez sa fiabilité",
			Code:    "CUSTOM_BUILDPACK_SECURITY",
		})
	}

	// Vérification de compatibilité avec l'application
	language := buildpack.GetLanguage()
	if language != "unknown" {
		// Vérifier la cohérence avec les processus
		hasMatchingProcess := false
		for _, process := range app.Config().Processes {
			if s.isLanguageCompatible(language, process.Command) {
				hasMatchingProcess = true
				break
			}
		}

		if !hasMatchingProcess {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "buildpack",
				Message: fmt.Sprintf("Le buildpack %s pourrait ne pas être compatible avec les processus configurés", language),
				Code:    "BUILDPACK_COMPATIBILITY_WARNING",
			})
		}
	}
}

// validateEnvironmentConsistency valide la cohérence avec l'environnement
func (s *DeploymentValidationService) validateEnvironmentConsistency(deployCtx *DeploymentContext, result *ValidationResult) {
	if deployCtx.Environment == "" {
		return
	}

	app := deployCtx.Application
	env := strings.ToLower(deployCtx.Environment)

	// Vérifications spécifiques à l'environnement de production
	if env == "production" {
		// Vérifier les domaines de production
		hasProductionDomain := false
		for _, domain := range app.Config().Domains {
			if !domain.IsLocalhost() && !domain.IsIP() {
				hasProductionDomain = true
				break
			}
		}

		if !hasProductionDomain {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "domains",
				Message: "Aucun domaine de production configuré",
				Code:    "NO_PRODUCTION_DOMAIN",
			})
		}

		// Vérifier les variables d'environnement sensibles
		envVars := app.Config().EnvironmentVars
		if _, hasDebug := envVars["DEBUG"]; hasDebug {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "environment_vars.DEBUG",
				Message: "Variable DEBUG détectée en production",
				Code:    "DEBUG_IN_PRODUCTION",
			})
		}
	}
}

// validateResourceAvailability valide la disponibilité des ressources
func (s *DeploymentValidationService) validateResourceAvailability(deployCtx *DeploymentContext, result *ValidationResult) {
	app := deployCtx.Application

	// Calculer les ressources totales requises
	totalInstances := 0
	for _, process := range app.Config().Processes {
		totalInstances += process.Scale
	}

	if totalInstances > 20 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "resources",
			Message: fmt.Sprintf("Nombre élevé d'instances requises: %d", totalInstances),
			Code:    "HIGH_RESOURCE_USAGE",
		})
	}

	// Vérifier les limites de ressources
	if limits := app.Config().ResourceLimits; limits != nil {
		if limits.Memory != "" && strings.HasSuffix(limits.Memory, "G") {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "resources.memory",
				Message: "Limite mémoire élevée configurée",
				Code:    "HIGH_MEMORY_LIMIT",
			})
		}
	}
}

// validateDeploymentConsistency vérifie la cohérence après déploiement
func (s *DeploymentValidationService) validateDeploymentConsistency(deployment *Deployment, app *Application, result *ValidationResult) {
	// Vérifier que l'état de l'application correspond au déploiement
	if app.State() != StateDeployed && app.State() != StateRunning {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "application.state",
			Message: "L'état de l'application ne correspond pas au déploiement réussi",
			Code:    "STATE_DEPLOYMENT_MISMATCH",
		})
	}

	// Vérifier la cohérence des références Git
	if app.GitRef() != nil && deployment.GitRef != app.GitRefString() {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "git_ref",
			Message: "Incohérence entre la référence Git du déploiement et de l'application",
			Code:    "GIT_REF_MISMATCH",
		})
	}
}

// validateApplicationHealth effectue des vérifications de santé basiques
func (s *DeploymentValidationService) validateApplicationHealth(app *Application, result *ValidationResult) {
	// Vérifier que l'application a des processus actifs
	hasActiveProcesses := false
	for _, process := range app.Config().Processes {
		if process.Scale > 0 {
			hasActiveProcesses = true
			break
		}
	}

	if !hasActiveProcesses {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "application.processes",
			Message: "Aucun processus actif détecté après le déploiement",
			Code:    "NO_ACTIVE_PROCESSES",
		})
	}
}

// Helper methods

func (s *DeploymentValidationService) mergeResults(target, source *ValidationResult) {
	target.Errors = append(target.Errors, source.Errors...)
	target.Warnings = append(target.Warnings, source.Warnings...)
	if !source.IsValid {
		target.IsValid = false
	}
}

func (s *DeploymentValidationService) isOldDeployment(deployment *Deployment) bool {
	if deployment.CreatedAt.IsZero() {
		return false
	}
	return time.Since(deployment.CreatedAt) > 30*24*time.Hour // Plus de 30 jours
}

func (s *DeploymentValidationService) isLanguageCompatible(language, command string) bool {
	command = strings.ToLower(command)
	switch language {
	case "node":
		return strings.Contains(command, "node") || strings.Contains(command, "npm") || strings.Contains(command, "yarn")
	case "python":
		return strings.Contains(command, "python") || strings.Contains(command, "pip") || strings.Contains(command, "gunicorn")
	case "ruby":
		return strings.Contains(command, "ruby") || strings.Contains(command, "bundle") || strings.Contains(command, "rails")
	case "java":
		return strings.Contains(command, "java") || strings.Contains(command, "gradle") || strings.Contains(command, "maven")
	case "php":
		return strings.Contains(command, "php") || strings.Contains(command, "composer")
	case "go":
		return strings.Contains(command, "go ") || strings.Contains(command, "./") || strings.Contains(command, "main")
	default:
		return false
	}
}

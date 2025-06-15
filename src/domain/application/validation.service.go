package application

import (
	"context"
	"fmt"
	"strings"
)

// ValidationService service domain pour valider les applications
type ValidationService struct{}

// NewValidationService crée un nouveau service de validation
func NewValidationService() *ValidationService {
	return &ValidationService{}
}

// ValidationResult résultat d'une validation
type ValidationResult struct {
	IsValid  bool
	Errors   []ValidationError
	Warnings []ValidationWarning
}

// ValidationError erreur de validation
type ValidationError struct {
	Field   string
	Message string
	Code    string
}

// ValidationWarning avertissement de validation
type ValidationWarning struct {
	Field   string
	Message string
	Code    string
}

// ValidateApplication valide une application complète
func (s *ValidationService) ValidateApplication(ctx context.Context, app *Application) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// Validation du nom
	s.validateApplicationName(app.Name(), result)

	// Validation de la configuration
	s.validateApplicationConfig(app.Config(), result)

	// Validation de l'état
	s.validateApplicationState(app, result)

	// Validation des processus
	s.validateProcesses(app.Config().Processes, result)

	// Validation des domaines
	s.validateDomains(app.Config().Domains, result)

	// Validation des variables d'environnement
	s.validateEnvironmentVariables(app.Config().EnvironmentVars, result)

	return result
}

// ValidateDeployment valide un déploiement
func (s *ValidationService) ValidateDeployment(ctx context.Context, app *Application, gitRef *GitRef, buildpack *BuildpackName) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// L'application doit exister
	if app == nil {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "application",
			Message: "L'application est requise pour le déploiement",
			Code:    "APP_REQUIRED",
		})
		return result
	}

	// Validation de la référence Git
	if gitRef != nil {
		s.validateGitRefForDeployment(gitRef, result)
	}

	// Validation du buildpack
	if buildpack != nil {
		s.validateBuildpackForDeployment(buildpack, app, result)
	}

	// Vérification de l'état de l'application
	if app.State() == StateError {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "state",
			Message: "L'application est en erreur, le déploiement pourrait échouer",
			Code:    "APP_ERROR_STATE",
		})
	}

	return result
}

// ValidateScale valide une opération de scaling
func (s *ValidationService) ValidateScale(ctx context.Context, app *Application, processType ProcessType, scale int) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// Validation du scale
	if scale < 0 {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "scale",
			Message: "Le nombre d'instances ne peut pas être négatif",
			Code:    "INVALID_SCALE",
		})
	}

	if scale > 50 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "scale",
			Message: "Un nombre élevé d'instances peut impacter les performances",
			Code:    "HIGH_SCALE_WARNING",
		})
	}

	// Vérification que le processus existe
	if _, exists := app.Config().Processes[processType]; !exists && scale > 0 {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "process_type",
			Message: fmt.Sprintf("Le type de processus %s n'existe pas", processType),
			Code:    "PROCESS_NOT_FOUND",
		})
	}

	return result
}

// validateApplicationName valide le nom de l'application
func (s *ValidationService) validateApplicationName(name string, result *ValidationResult) {
	if name == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "name",
			Message: "Le nom de l'application ne peut pas être vide",
			Code:    "NAME_EMPTY",
		})
		return
	}

	if len(name) > 63 {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "name",
			Message: "Le nom de l'application ne peut pas dépasser 63 caractères",
			Code:    "NAME_TOO_LONG",
		})
	}

	// Validation du format DNS
	if !strings.Contains(name, "-") && len(name) > 15 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "name",
			Message: "Un nom plus court avec des tirets améliore la lisibilité",
			Code:    "NAME_FORMAT_SUGGESTION",
		})
	}
}

// validateApplicationConfig valide la configuration de l'application
func (s *ValidationService) validateApplicationConfig(config *ApplicationConfig, result *ValidationResult) {
	if config == nil {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "config",
			Message: "La configuration de l'application est requise",
			Code:    "CONFIG_REQUIRED",
		})
		return
	}

	// Validation des limites de ressources
	if config.ResourceLimits != nil {
		s.validateResourceLimits(config.ResourceLimits, result)
	}
}

// validateApplicationState valide l'état de l'application
func (s *ValidationService) validateApplicationState(app *Application, result *ValidationResult) {
	// Vérifications de cohérence d'état
	if app.IsRunning() && app.LastDeploy() == nil {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "state",
			Message: "L'application est marquée comme en cours d'exécution mais n'a jamais été déployée",
			Code:    "STATE_INCONSISTENCY",
		})
	}

	if app.State() == StateDeployed && len(app.Config().Processes) == 0 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "processes",
			Message: "L'application est déployée mais n'a aucun processus configuré",
			Code:    "NO_PROCESSES_CONFIGURED",
		})
	}
}

// validateProcesses valide les processus de l'application
func (s *ValidationService) validateProcesses(processes map[ProcessType]*Process, result *ValidationResult) {
	if len(processes) == 0 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "processes",
			Message: "Aucun processus configuré, l'application pourrait ne pas démarrer",
			Code:    "NO_PROCESSES",
		})
		return
	}

	// Vérifier qu'il y a au moins un processus web
	if _, hasWeb := processes[ProcessTypeWeb]; !hasWeb {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "processes",
			Message: "Aucun processus web configuré, l'application ne sera pas accessible via HTTP",
			Code:    "NO_WEB_PROCESS",
		})
	}

	// Validation de chaque processus
	for processType, process := range processes {
		if process.Command == "" {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("processes.%s.command", processType),
				Message: "La commande du processus ne peut pas être vide",
				Code:    "EMPTY_PROCESS_COMMAND",
			})
		}

		if process.Scale < 0 {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("processes.%s.scale", processType),
				Message: "Le nombre d'instances ne peut pas être négatif",
				Code:    "NEGATIVE_PROCESS_SCALE",
			})
		}
	}
}

// validateDomains valide les domaines de l'application
func (s *ValidationService) validateDomains(domains []string, result *ValidationResult) {
	seenDomains := make(map[string]bool)

	for i, domain := range domains {
		// Vérifier les doublons
		if seenDomains[domain] {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   fmt.Sprintf("domains[%d]", i),
				Message: fmt.Sprintf("Le domaine %s est dupliqué", domain),
				Code:    "DUPLICATE_DOMAIN",
			})
		}
		seenDomains[domain] = true

		// Valider le format du domaine
		if _, err := NewDomain(domain); err != nil {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("domains[%d]", i),
				Message: fmt.Sprintf("Domaine invalide: %v", err),
				Code:    "INVALID_DOMAIN",
			})
		}
	}
}

// validateEnvironmentVariables valide les variables d'environnement
func (s *ValidationService) validateEnvironmentVariables(envVars map[string]string, result *ValidationResult) {
	sensitiveKeys := []string{"PASSWORD", "SECRET", "KEY", "TOKEN", "API_KEY"}

	for key, value := range envVars {
		// Vérifier les clés vides
		if key == "" {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "environment_vars",
				Message: "Les clés de variables d'environnement ne peuvent pas être vides",
				Code:    "EMPTY_ENV_KEY",
			})
		}

		// Avertissement pour les valeurs sensibles en clair
		for _, sensitiveKey := range sensitiveKeys {
			if strings.Contains(strings.ToUpper(key), sensitiveKey) && value != "" {
				result.Warnings = append(result.Warnings, ValidationWarning{
					Field:   fmt.Sprintf("environment_vars.%s", key),
					Message: "Variable sensible détectée, assurez-vous qu'elle est sécurisée",
					Code:    "SENSITIVE_ENV_VAR",
				})
				break
			}
		}
	}
}

// validateResourceLimits valide les limites de ressources
func (s *ValidationService) validateResourceLimits(limits *ResourceLimits, result *ValidationResult) {
	// Validation de la mémoire
	if limits.Memory != "" {
		if !strings.HasSuffix(limits.Memory, "M") && !strings.HasSuffix(limits.Memory, "G") {
			result.IsValid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "resource_limits.memory",
				Message: "Le format de la limite mémoire doit se terminer par M ou G",
				Code:    "INVALID_MEMORY_FORMAT",
			})
		}
	}

	// Validation du CPU
	if limits.CPU != "" {
		if !strings.Contains(limits.CPU, ".") && limits.CPU != "1" {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "resource_limits.cpu",
				Message: "Le format CPU recommandé utilise des décimales (ex: 0.5, 1.0)",
				Code:    "CPU_FORMAT_SUGGESTION",
			})
		}
	}
}

// validateGitRefForDeployment valide une référence Git pour le déploiement
func (s *ValidationService) validateGitRefForDeployment(gitRef *GitRef, result *ValidationResult) {
	if gitRef.IsSHA() && len(gitRef.Value()) < 7 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "git_ref",
			Message: "SHA Git très court, pourrait être ambigu",
			Code:    "SHORT_SHA_WARNING",
		})
	}
}

// validateBuildpackForDeployment valide un buildpack pour le déploiement
func (s *ValidationService) validateBuildpackForDeployment(buildpack *BuildpackName, app *Application, result *ValidationResult) {
	if !buildpack.IsOfficial() && !buildpack.IsURL() {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "buildpack",
			Message: "Buildpack non-officiel détecté, vérifiez sa compatibilité",
			Code:    "CUSTOM_BUILDPACK_WARNING",
		})
	}

	// Vérification de cohérence avec les processus existants
	detectedLang := buildpack.GetLanguage()
	if detectedLang != "unknown" {
		hasMatchingProcess := false
		for _, process := range app.Config().Processes {
			if strings.Contains(strings.ToLower(process.Command), detectedLang) {
				hasMatchingProcess = true
				break
			}
		}

		if !hasMatchingProcess {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "buildpack",
				Message: fmt.Sprintf("Le buildpack %s ne correspond pas aux processus configurés", detectedLang),
				Code:    "BUILDPACK_PROCESS_MISMATCH",
			})
		}
	}
}

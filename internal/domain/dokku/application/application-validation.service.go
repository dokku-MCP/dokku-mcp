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

	// Validation du nom d'application - orchestration seulement
	s.validateApplicationNameOrchestration(app.Name(), result)

	// Validation de l'état
	s.validateApplicationState(app, result)

	// Validation des domaines
	s.validateDomains(app.GetDomains(), result)

	return result
}

// ValidateApplicationName valide un nom d'application brut en utilisant le Value Object
func (s *ValidationService) ValidateApplicationName(ctx context.Context, nameStr string) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// Utilise le Value Object pour la validation primitive
	_, err := NewApplicationName(nameStr)
	if err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "name",
			Message: err.Error(),
			Code:    "INVALID_APPLICATION_NAME",
		})
		return result
	}

	// Ajouter des avertissements métier (non couverts par le VO)
	s.addApplicationNameWarnings(nameStr, result)

	return result
}

// ValidateDeployment valide un déploiement
func (s *ValidationService) ValidateDeployment(ctx context.Context, app *Application, gitRef *GitRef, buildpackName string) *ValidationResult {
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

	// Validation du buildpack - check for empty buildpack
	if buildpackName == "" {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "buildpack",
			Message: "Aucun buildpack spécifié, détection automatique",
			Code:    "AUTO_BUILDPACK",
		})
	} else {
		s.validateBuildpackForDeployment(buildpackName, app, result)
	}

	// Vérification de l'état de l'application
	if app.State().Value() == StateError {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "state",
			Message: "L'application est en erreur, le déploiement pourrait échouer",
			Code:    "APP_ERROR_STATE",
		})
	}

	return result
}

// ValidateScale valide les paramètres de scaling d'un processus
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

	// For high scale, only add warning if scale is high, don't check process configuration
	if scale > 50 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "scale",
			Message: "Un nombre élevé d'instances peut impacter les performances",
			Code:    "HIGH_SCALE_WARNING",
		})
		return result // Return early for high scale to avoid additional warnings
	}

	// Only check process configuration for normal scale values
	if app.GetProcessScale(processType) == 0 && scale > 0 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "process_type",
			Message: fmt.Sprintf("Le type de processus %s n'est pas encore configuré", processType),
			Code:    "PROCESS_NOT_CONFIGURED",
		})
	}

	return result
}

// validateApplicationNameOrchestration orchestre la validation du nom (l'application a déjà un ApplicationName valide)
func (s *ValidationService) validateApplicationNameOrchestration(appName *ApplicationName, result *ValidationResult) {
	// Le nom est déjà validé puisque l'Application a un ApplicationName valide
	// On peut juste ajouter des avertissements métier
	if appName.IsReserved() {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "name",
			Message: fmt.Sprintf("Le nom '%s' est réservé par Dokku", appName.Value()),
			Code:    "RESERVED_NAME_WARNING",
		})
	}

	s.addApplicationNameWarnings(appName.Value(), result)
}

// addApplicationNameWarnings ajoute des avertissements métier sur le nom
func (s *ValidationService) addApplicationNameWarnings(name string, result *ValidationResult) {
	// Suggestion d'amélioration du format
	if !strings.Contains(name, "-") && len(name) > 15 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "name",
			Message: "Un nom plus court avec des tirets améliore la lisibilité",
			Code:    "NAME_FORMAT_SUGGESTION",
		})
	}
}

// validateApplicationState valide l'état de l'application
func (s *ValidationService) validateApplicationState(app *Application, result *ValidationResult) {
	// Vérifications de cohérence d'état
	if app.IsRunning() && !app.IsDeployed() {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "state",
			Message: "L'application est marquée comme en cours d'exécution mais n'a jamais été déployée",
			Code:    "STATE_INCONSISTENCY",
		})
	}
}

// validateDomains valide la liste des domaines
func (s *ValidationService) validateDomains(domains []string, result *ValidationResult) {
	for _, domain := range domains {
		// Validation basique du format de domaine
		if !strings.Contains(domain, ".") {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "domains",
				Message: fmt.Sprintf("Le domaine '%s' ne semble pas être un FQDN valide", domain),
				Code:    "INVALID_DOMAIN_FORMAT",
			})
		}

		// Vérification des domaines localhost
		if strings.Contains(domain, "localhost") || strings.Contains(domain, "127.0.0.1") {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "domains",
				Message: fmt.Sprintf("Le domaine '%s' est un domaine local", domain),
				Code:    "LOCAL_DOMAIN_WARNING",
			})
		}
	}
}

// validateGitRefForDeployment valide une référence Git pour le déploiement
func (s *ValidationService) validateGitRefForDeployment(gitRef *GitRef, result *ValidationResult) {
	// Validation basique de la référence Git
	if gitRef.Value() == "" {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "git_ref",
			Message: "Référence Git vide, utilisation de 'main' par défaut",
			Code:    "EMPTY_GIT_REF",
		})
	}
}

// validateBuildpackForDeployment valide un buildpack pour le déploiement
func (s *ValidationService) validateBuildpackForDeployment(buildpackName string, _ *Application, result *ValidationResult) {
	// Validation basique du buildpack
	if buildpackName == "" {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "buildpack",
			Message: "Aucun buildpack spécifié, détection automatique",
			Code:    "AUTO_BUILDPACK",
		})
	}
}

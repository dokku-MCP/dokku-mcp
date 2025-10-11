package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/dokku-mcp/dokku-mcp/internal/shared"
	"github.com/dokku-mcp/dokku-mcp/internal/shared/process"
)

// ValidationService is a domain service for validating applications
type ValidationService struct{}

// NewValidationService creates a new validation service
func NewValidationService() *ValidationService {
	return &ValidationService{}
}

// ValidationResult is the result of a validation
type ValidationResult struct {
	IsValid  bool
	Errors   []ValidationError
	Warnings []ValidationWarning
}

// ValidationError is a validation error
type ValidationError struct {
	Field   string
	Message string
	Code    string
}

// ValidationWarning is a validation warning
type ValidationWarning struct {
	Field   string
	Message string
	Code    string
}

// ValidateApplication validates a complete application
func (s *ValidationService) ValidateApplication(ctx context.Context, app *Application) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// Validate application name - orchestration only
	s.validateApplicationNameOrchestration(app.Name(), result)

	// Validate state
	s.validateApplicationState(app, result)

	// Validate domains
	s.validateDomains(app.GetDomains(), result)

	return result
}

// ValidateApplicationName validates a raw application name using the Value Object
func (s *ValidationService) ValidateApplicationName(ctx context.Context, nameStr string) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// Use Value Object for primitive validation
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

	// Add business warnings (not covered by VO)
	s.addApplicationNameWarnings(nameStr, result)

	return result
}

// ValidateDeployment validates a deployment
func (s *ValidationService) ValidateDeployment(ctx context.Context, app *Application, gitRef *shared.GitRef, buildpackName string) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// The application must exist
	if app == nil {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "application",
			Message: "Application is required for deployment",
			Code:    "APP_REQUIRED",
		})
		return result
	}

	// Validate Git reference
	if gitRef != nil {
		s.validateGitRefForDeployment(gitRef, result)
	}

	// Validate buildpack - check for empty buildpack
	if buildpackName == "" {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "buildpack",
			Message: "No buildpack specified, auto-detection will be used",
			Code:    "AUTO_BUILDPACK",
		})
	} else {
		s.validateBuildpackForDeployment(buildpackName, app, result)
	}

	// Check application state
	if app.State().Value() == StateError {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "state",
			Message: "Application is in an error state, deployment might fail",
			Code:    "APP_ERROR_STATE",
		})
	}

	return result
}

// ValidateScale validates the scaling parameters of a process
func (s *ValidationService) ValidateScale(ctx context.Context, app *Application, processType process.ProcessType, scale int) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   make([]ValidationError, 0),
		Warnings: make([]ValidationWarning, 0),
	}

	// Validate scale
	if scale < 0 {
		result.IsValid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "scale",
			Message: "Number of instances cannot be negative",
			Code:    "INVALID_SCALE",
		})
	}

	// For high scale, only add warning if scale is high, don't check process configuration
	if scale > 50 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "scale",
			Message: "A high number of instances may impact performance",
			Code:    "HIGH_SCALE_WARNING",
		})
		return result // Return early for high scale to avoid additional warnings
	}

	// Only check process configuration for normal scale values
	if app.GetProcessScale(processType) == 0 && scale > 0 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "process_type",
			Message: fmt.Sprintf("Process type %s is not yet configured", processType),
			Code:    "PROCESS_NOT_CONFIGURED",
		})
	}

	return result
}

// validateApplicationNameOrchestration orchestrates name validation (application already has a valid ApplicationName)
func (s *ValidationService) validateApplicationNameOrchestration(appName *ApplicationName, result *ValidationResult) {
	// The name is already validated since the Application has a valid ApplicationName
	// We can just add business warnings
	if appName.IsReserved() {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "name",
			Message: fmt.Sprintf("Name '%s' is reserved by Dokku", appName.Value()),
			Code:    "RESERVED_NAME_WARNING",
		})
	}

	s.addApplicationNameWarnings(appName.Value(), result)
}

// addApplicationNameWarnings adds business warnings about the name
func (s *ValidationService) addApplicationNameWarnings(name string, result *ValidationResult) {
	// Suggest format improvement
	if !strings.Contains(name, "-") && len(name) > 15 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "name",
			Message: "A shorter name with hyphens improves readability",
			Code:    "NAME_FORMAT_SUGGESTION",
		})
	}
}

// validateApplicationState validates the application state
func (s *ValidationService) validateApplicationState(app *Application, result *ValidationResult) {
	// Check for state consistency
	if app.IsRunning() && !app.IsDeployed() {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "state",
			Message: "Application is marked as running but has never been deployed",
			Code:    "STATE_INCONSISTENCY",
		})
	}
}

// validateDomains validates the list of domains
func (s *ValidationService) validateDomains(domains []string, result *ValidationResult) {
	for _, domain := range domains {
		// Basic validation of domain format
		if !strings.Contains(domain, ".") {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "domains",
				Message: fmt.Sprintf("Domain '%s' does not appear to be a valid FQDN", domain),
				Code:    "INVALID_DOMAIN_FORMAT",
			})
		}

		// Check for localhost domains
		if strings.Contains(domain, "localhost") || strings.Contains(domain, "127.0.0.1") {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "domains",
				Message: fmt.Sprintf("Domain '%s' is a local domain", domain),
				Code:    "LOCAL_DOMAIN_WARNING",
			})
		}
	}
}

// validateGitRefForDeployment validates a Git reference for deployment
func (s *ValidationService) validateGitRefForDeployment(gitRef *shared.GitRef, result *ValidationResult) {
	// Basic validation of Git reference
	if gitRef.Value() == "" {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "git_ref",
			Message: "Empty Git reference, 'main' will be used by default",
			Code:    "EMPTY_GIT_REF",
		})
	}
}

// validateBuildpackForDeployment validates a buildpack for deployment
func (s *ValidationService) validateBuildpackForDeployment(buildpackName string, _ *Application, result *ValidationResult) {
	// Basic validation of buildpack
	if buildpackName == "" {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "buildpack",
			Message: "No buildpack specified, auto-detection will be used",
			Code:    "AUTO_BUILDPACK",
		})
	}
}

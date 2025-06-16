package dokku

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	app_domain "github.com/alex-galey/dokku-mcp/internal/domain/dokku/application"
)

// applicationRepository implémente app_domain.ApplicationRepository avec Dokku
type applicationRepository struct {
	client DokkuClient
	logger *slog.Logger
}

// NewApplicationRepository crée un nouveau repository d'application
func NewApplicationRepository(client DokkuClient, logger *slog.Logger) app_domain.ApplicationRepository {
	return &applicationRepository{
		client: client,
		logger: logger,
	}
}

// GetAll récupère toutes les applications
func (r *applicationRepository) GetAll(ctx context.Context) ([]*app_domain.Application, error) {
	r.logger.Debug("Récupération de toutes les applications")

	appNames, err := r.client.GetApplications(ctx)
	if err != nil {
		return nil, fmt.Errorf("échec de récupération des noms d'applications: %w", err)
	}

	applications := make([]*app_domain.Application, 0, len(appNames))

	for _, appName := range appNames {
		appNameVO, err := app_domain.NewApplicationName(appName)
		if err != nil {
			r.logger.Warn("Nom d'application invalide, ignoré",
				"erreur", err,
				"nom_app", appName)
			continue
		}

		app, err := r.GetByName(ctx, appNameVO)
		if err != nil {
			r.logger.Warn("Échec de récupération de l'application",
				"erreur", err,
				"nom_app", appName)
			continue
		}
		applications = append(applications, app)
	}

	r.logger.Debug("Applications récupérées avec succès",
		"nombre", len(applications))
	return applications, nil
}

// GetByName récupère une application par son nom
func (r *applicationRepository) GetByName(ctx context.Context, name *app_domain.ApplicationName) (*app_domain.Application, error) {
	r.logger.Debug("Récupération d'application par nom",
		"nom_app", name.Value())

	// Créer l'entité application de base
	app, err := app_domain.NewApplication(name.Value())
	if err != nil {
		return nil, fmt.Errorf("échec de création de l'entité application: %w", err)
	}

	// Vérifier si l'application existe dans Dokku
	exists, err := r.Exists(ctx, name)
	if err != nil {
		r.logger.Warn("Impossible de vérifier l'existence de l'application",
			"erreur", err,
			"nom_app", name.Value())
	} else if !exists {
		r.logger.Warn("L'application n'existe pas dans Dokku",
			"nom_app", name.Value())
		// Retourner l'application de base même si elle n'existe pas vraiment
		r.logger.Debug("Application récupérée avec informations minimales",
			"nom_app", name.Value())
		return app, nil
	}

	// Récupérer les informations détaillées si possible
	info, err := r.client.GetApplicationInfo(ctx, name.Value())
	if err != nil {
		r.logger.Warn("Échec de récupération des informations détaillées - utilisation des informations de base",
			"erreur", err,
			"nom_app", name.Value())

		// Essayer de récupérer des informations de base via apps:report si disponible
		if reportInfo, reportErr := r.tryGetBasicApplicationInfo(ctx, name.Value()); reportErr == nil {
			r.logger.Debug("Informations de base récupérées via apps:report",
				"nom_app", name.Value())
			if err := r.updateApplicationFromInfo(app, reportInfo, make(map[string]string)); err != nil {
				r.logger.Warn("Échec de mise à jour depuis apps:report",
					"erreur", err,
					"nom_app", name.Value())
			}
		}

		r.logger.Debug("Application récupérée avec informations de base",
			"nom_app", name.Value())
		return app, nil
	}

	// Récupérer la configuration
	config, err := r.client.GetApplicationConfig(ctx, name.Value())
	if err != nil {
		r.logger.Warn("Échec de récupération de la configuration - utilisation d'une configuration vide",
			"erreur", err,
			"nom_app", name.Value())
		config = make(map[string]string)
	}

	// Mettre à jour l'application avec les informations récupérées
	if err := r.updateApplicationFromInfo(app, info, config); err != nil {
		r.logger.Warn("Échec de mise à jour de l'application depuis les informations Dokku",
			"erreur", err,
			"nom_app", name.Value())
	}

	r.logger.Debug("Application récupérée avec succès",
		"nom_app", name.Value())
	return app, nil
}

// Save sauvegarde une application
func (r *applicationRepository) Save(ctx context.Context, app *app_domain.Application) error {
	r.logger.Debug("Sauvegarde de l'application",
		"nom_app", app.Name().Value())

	exists, err := r.Exists(ctx, app.Name())
	if err != nil {
		return fmt.Errorf("échec de vérification de l'existence de l'application: %w", err)
	}

	if !exists {
		_, err := r.client.ExecuteCommand(ctx, "apps:create", []string{app.Name().Value()})
		if err != nil {
			return fmt.Errorf("échec de création de l'application: %w", err)
		}
	}

	// Mettre à jour la configuration si elle existe
	if config := app.Configuration(); config != nil {
		configMap := r.extractEnvironmentVars(config)
		if len(configMap) > 0 {
			if err := r.client.SetApplicationConfig(ctx, app.Name().Value(), configMap); err != nil {
				return fmt.Errorf("échec de mise à jour de la configuration: %w", err)
			}
		}
	}

	r.logger.Debug("Application sauvegardée avec succès",
		"nom_app", app.Name().Value())
	return nil
}

// Delete supprime une application
func (r *applicationRepository) Delete(ctx context.Context, name *app_domain.ApplicationName) error {
	r.logger.Debug("Suppression de l'application",
		"nom_app", name.Value())

	_, err := r.client.ExecuteCommand(ctx, "apps:destroy", []string{name.Value(), "--force"})
	if err != nil {
		return fmt.Errorf("échec de suppression de l'application: %w", err)
	}

	r.logger.Debug("Application supprimée avec succès",
		"nom_app", name.Value())
	return nil
}

// Exists vérifie si une application existe
func (r *applicationRepository) Exists(ctx context.Context, name *app_domain.ApplicationName) (bool, error) {
	r.logger.Debug("Vérification de l'existence de l'application",
		"nom_app", name.Value())

	_, err := r.client.ExecuteCommand(ctx, "apps:exists", []string{name.Value()})
	if err != nil {
		return false, nil
	}

	return true, nil
}

// List récupère une liste paginée d'applications
func (r *applicationRepository) List(ctx context.Context, offset, limit int) ([]*app_domain.Application, int, error) {
	r.logger.Debug("Récupération d'une liste paginée d'applications",
		"offset", offset,
		"limit", limit)

	allApps, err := r.GetAll(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("échec de récupération de toutes les applications: %w", err)
	}

	total := len(allApps)

	start := min(offset, total)
	end := min(start+limit, total)

	pagedApps := allApps[start:end]

	r.logger.Debug("Liste paginée récupérée avec succès",
		"total", total,
		"retourné", len(pagedApps))

	return pagedApps, total, nil
}

// GetByState récupère les applications par état
func (r *applicationRepository) GetByState(ctx context.Context, state *app_domain.ApplicationState) ([]*app_domain.Application, error) {
	r.logger.Debug("Récupération d'applications par état",
		"état", state.Value())

	allApps, err := r.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("échec de récupération de toutes les applications: %w", err)
	}

	var filteredApps []*app_domain.Application
	for _, app := range allApps {
		if app.State().Value() == state.Value() {
			filteredApps = append(filteredApps, app)
		}
	}

	r.logger.Debug("Applications récupérées par état",
		"état", state.Value(),
		"nombre", len(filteredApps))

	return filteredApps, nil
}

// Méthodes utilitaires privées
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// updateApplicationFromInfo met à jour l'application avec les informations récupérées
func (r *applicationRepository) updateApplicationFromInfo(app *app_domain.Application, info map[string]interface{}, config map[string]string) error {
	// Appliquer les variables d'environnement
	for key, value := range config {
		if err := app.SetEnvironmentVariable(key, value); err != nil {
			r.logger.Warn("Échec de définition de la variable d'environnement",
				"clé", key,
				"erreur", err)
		}
	}

	// Traiter les processus si présents dans les informations
	if processesStr, ok := info["ps.scale"].(string); ok && processesStr != "" {
		r.parseProcesses(app, processesStr)
	}

	// Traiter les domaines si présents
	if domainsStr, ok := info["domains"].(string); ok && domainsStr != "" {
		domains := strings.Split(domainsStr, " ")
		for _, domain := range domains {
			if domain != "" {
				if err := app.AddDomain(domain); err != nil {
					r.logger.Warn("Échec d'ajout du domaine",
						"domaine", domain,
						"erreur", err)
				}
			}
		}
	}

	return nil
}

// parseProcesses analyse et ajoute les processus depuis une chaîne
func (r *applicationRepository) parseProcesses(app *app_domain.Application, processesStr string) {
	processes := strings.Fields(processesStr)
	for _, process := range processes {
		parts := strings.Split(process, ":")
		if len(parts) == 2 {
			processType := parts[0]
			scaleStr := parts[1]

			scale, err := strconv.Atoi(scaleStr)
			if err != nil {
				r.logger.Warn("Échec de parsing de l'échelle du processus",
					"processus", process,
					"erreur", err)
				continue
			}

			// Convert string to ProcessType directly
			processTypeVO := app_domain.ProcessType(processType)

			// Ajouter le processus s'il n'existe pas déjà
			if err := app.AddProcess(processTypeVO, "", scale); err != nil {
				r.logger.Warn("Échec d'ajout du processus",
					"type", processType,
					"erreur", err)
			}
		}
	}
}

// tryGetBasicApplicationInfo essaie de récupérer des informations de base
func (r *applicationRepository) tryGetBasicApplicationInfo(ctx context.Context, appName string) (map[string]interface{}, error) {
	output, err := r.client.ExecuteCommand(ctx, "apps:report", []string{appName})
	if err != nil {
		return nil, fmt.Errorf("échec d'exécution d'apps:report: %w", err)
	}

	// Parser la sortie apps:report pour extraire les informations de base
	info := make(map[string]interface{})
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				info[key] = value
			}
		}
	}

	return info, nil
}

// extractEnvironmentVars extrait les variables d'environnement de la configuration
func (r *applicationRepository) extractEnvironmentVars(config *app_domain.ApplicationConfiguration) map[string]string {
	// TODO: Implémenter l'extraction réelle des variables d'environnement
	// selon l'interface de ApplicationConfiguration
	return make(map[string]string)
}

// GetByDomain récupère les applications par domaine
func (r *applicationRepository) GetByDomain(ctx context.Context, domain string) ([]*app_domain.Application, error) {
	r.logger.Debug("Récupération d'applications par domaine",
		"domaine", domain)

	allApps, err := r.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("échec de récupération de toutes les applications: %w", err)
	}

	var filteredApps []*app_domain.Application
	for _, app := range allApps {
		if app.HasDomain(domain) {
			filteredApps = append(filteredApps, app)
		}
	}

	r.logger.Debug("Applications récupérées par domaine",
		"domaine", domain,
		"nombre", len(filteredApps))

	return filteredApps, nil
}

// GetRunningApplications récupère les applications en cours d'exécution
func (r *applicationRepository) GetRunningApplications(ctx context.Context) ([]*app_domain.Application, error) {
	r.logger.Debug("Récupération des applications en cours d'exécution")

	runningState, err := app_domain.NewApplicationState(app_domain.StateRunning)
	if err != nil {
		return nil, fmt.Errorf("échec de création de l'état en cours d'exécution: %w", err)
	}

	return r.GetByState(ctx, runningState)
}

// GetApplicationsWithBuildpack récupère les applications avec un buildpack spécifique
func (r *applicationRepository) GetApplicationsWithBuildpack(ctx context.Context, buildpack string) ([]*app_domain.Application, error) {
	r.logger.Debug("Récupération d'applications par buildpack",
		"buildpack", buildpack)

	allApps, err := r.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("échec de récupération de toutes les applications: %w", err)
	}

	var filteredApps []*app_domain.Application
	// TODO: Vérifier le buildpack de l'application selon l'interface réelle
	// Pour l'instant, on retourne toutes les applications
	filteredApps = append(filteredApps, allApps...)

	r.logger.Debug("Applications récupérées par buildpack",
		"buildpack", buildpack,
		"nombre", len(filteredApps))

	return filteredApps, nil
}

// GetRecentlyDeployed récupère les applications récemment déployées
func (r *applicationRepository) GetRecentlyDeployed(ctx context.Context, limit int) ([]*app_domain.Application, error) {
	r.logger.Debug("Récupération des applications récemment déployées",
		"limite", limit)

	allApps, err := r.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("échec de récupération de toutes les applications: %w", err)
	}

	// TODO: Trier par date de déploiement et limiter
	// Pour l'instant, on retourne les premières applications jusqu'à la limite
	if len(allApps) > limit {
		allApps = allApps[:limit]
	}

	r.logger.Debug("Applications récemment déployées récupérées",
		"nombre", len(allApps))

	return allApps, nil
}

// CountByState compte les applications par état
func (r *applicationRepository) CountByState(ctx context.Context) (map[app_domain.StateValue]int, error) {
	r.logger.Debug("Comptage des applications par état")

	allApps, err := r.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("échec de récupération de toutes les applications: %w", err)
	}

	counts := make(map[app_domain.StateValue]int)
	for _, app := range allApps {
		counts[app.State().Value()]++
	}

	r.logger.Debug("Comptage par état terminé",
		"états", len(counts))

	return counts, nil
}

// GetApplicationMetrics récupère les métriques des applications
func (r *applicationRepository) GetApplicationMetrics(ctx context.Context) (*app_domain.ApplicationMetrics, error) {
	r.logger.Debug("Récupération des métriques d'applications")

	allApps, err := r.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("échec de récupération de toutes les applications: %w", err)
	}

	counts, err := r.CountByState(ctx)
	if err != nil {
		return nil, fmt.Errorf("échec de comptage par état: %w", err)
	}

	metrics := &app_domain.ApplicationMetrics{
		TotalApplications:     len(allApps),
		RunningApplications:   counts[app_domain.StateRunning],
		StoppedApplications:   counts[app_domain.StateStopped],
		ErrorApplications:     counts[app_domain.StateError],
		DeployingApplications: counts[app_domain.StateDeploying],
		ApplicationsByState:   counts,
		MostUsedBuildpacks:    make(map[string]int),
		// TODO: Implémenter les métriques de déploiement
		TotalDeployments:      0,
		SuccessfulDeployments: 0,
		FailedDeployments:     0,
		AverageDeploymentTime: 0.0,
	}

	r.logger.Debug("Métriques d'applications récupérées")

	return metrics, nil
}

package dokku

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/alex-galey/dokku-mcp/src/domain/application"
)

type deploymentService struct {
	client DokkuClient
	logger *slog.Logger
}

func NewDeploymentService(client DokkuClient, logger *slog.Logger) application.DeploymentService {
	return &deploymentService{
		client: client,
		logger: logger,
	}
}

func (s *deploymentService) Deploy(ctx context.Context, appName string, options application.DeployOptions) (*application.Deployment, error) {
	s.logger.Info("Starting application deployment",
		"app_name", appName,
		"git_ref", options.GitRef)

	deployment, err := application.NewDeployment(appName, options.GitRef)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	deployment.Start()

	exists, err := s.checkApplicationExists(ctx, appName)
	if err != nil {
		deployment.Fail(fmt.Sprintf("Failed to check application existence: %v", err))
		return deployment, fmt.Errorf("failed to check application existence: %w", err)
	}

	if !exists {
		if err := s.createApplication(ctx, appName); err != nil {
			deployment.Fail(fmt.Sprintf("Failed to create application: %v", err))
			return deployment, fmt.Errorf("failed to create application: %w", err)
		}
		s.logger.Info("Application created successfully",
			"app_name", appName)
	}

	// Définir le buildpack si spécifié
	if options.BuildPack != "" {
		if err := s.setBuildpack(ctx, appName, options.BuildPack); err != nil {
			s.logger.Warn("Failed to set buildpack",
				"error", err)
		}
	}

	if err := s.performGitDeploy(ctx, appName, options.GitRef); err != nil {
		deployment.Fail(fmt.Sprintf("Failed to deploy from git: %v", err))
		return deployment, fmt.Errorf("failed to deploy from git: %w", err)
	}

	s.logger.Info("Git deployment completed successfully",
		"app_name", appName,
		"git_ref", options.GitRef,
		"deployment_id", deployment.ID)

	deployment.Complete()

	s.logger.Info("Deployment completed successfully",
		"app_name", appName,
		"deployment_id", deployment.ID,
		"duration", deployment.Duration())

	return deployment, nil
}

// Rollback effectue un rollback d'application à une version spécifique
func (s *deploymentService) Rollback(ctx context.Context, appName string, version string) error {
	s.logger.Info("Starting application rollback",
		"app_name", appName,
		"version", version)

	// Dans une vraie implémentation, ceci utiliserait les commandes Dokku appropriées
	// Par exemple: dokku git:from-archive app archive.tar.gz
	// Pour l'instant, nous simulons le rollback

	s.logger.Info("Rollback simulated - operation successful",
		"app_name", appName,
		"version", version)

	return nil
}

func (s *deploymentService) GetHistory(ctx context.Context, appName string) ([]*application.Deployment, error) {
	s.logger.Debug("Retrieving deployment history",
		"app_name", appName)

	// Vérifier d'abord si l'application existe
	exists, err := s.checkApplicationExists(ctx, appName)
	if err != nil {
		return nil, fmt.Errorf("failed to check application existence: %w", err)
	}
	if !exists {
		return []*application.Deployment{}, nil
	}

	deployments, err := s.parseDeploymentHistory(ctx, appName)
	if err != nil {
		s.logger.Warn("Failed to retrieve deployment history from events",
			"error", err,
			"app_name", appName)

		// Fallback: essayer de récupérer au moins le déploiement actuel
		return s.getCurrentDeploymentAsFallback(ctx, appName)
	}

	s.logger.Debug("Deployment history retrieved successfully",
		"app_name", appName,
		"number", len(deployments))

	return deployments, nil
}

func (s *deploymentService) performGitDeploy(ctx context.Context, appName string, gitRef string) error {
	args := []string{appName}
	if gitRef != "" {
		args = append(args, gitRef)
	}

	_, err := s.client.ExecuteCommand(ctx, "git:sync", args)
	if err != nil {
		return fmt.Errorf("failed to deploy git for %s: %w", appName, err)
	}

	return nil
}

func (s *deploymentService) checkApplicationExists(ctx context.Context, appName string) (bool, error) {
	_, err := s.client.ExecuteCommand(ctx, "apps:exists", []string{appName})
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (s *deploymentService) createApplication(ctx context.Context, appName string) error {
	_, err := s.client.ExecuteCommand(ctx, "apps:create", []string{appName})
	if err != nil {
		return fmt.Errorf("failed to create application %s: %w", appName, err)
	}
	return nil
}

func (s *deploymentService) setBuildpack(ctx context.Context, appName string, buildpack string) error {
	_, err := s.client.ExecuteCommand(ctx, "buildpacks:set", []string{appName, buildpack})
	if err != nil {
		return fmt.Errorf("failed to set buildpack %s for %s: %w", buildpack, appName, err)
	}
	return nil
}

func (s *deploymentService) parseDeploymentHistory(ctx context.Context, appName string) ([]*application.Deployment, error) {
	output, err := s.client.ExecuteCommand(ctx, "events", []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve Dokku events: %w", err)
	}

	return s.parseEventsOutput(string(output), appName)
}

func (s *deploymentService) parseEventsOutput(eventsOutput, appName string) ([]*application.Deployment, error) {
	lines := strings.Split(eventsOutput, "\n")
	deploymentMap := make(map[string]*application.Deployment)

	// Parse the event lines to find the deployments
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Search for events related to this application
		if !strings.Contains(line, appName) {
			continue
		}

		deployment := s.parseEventLine(line, appName)
		if deployment != nil {
			deploymentMap[deployment.ID] = deployment
		}
	}

	// Convert the map to a slice and sort by creation date (most recent first)
	deployments := make([]*application.Deployment, 0, len(deploymentMap))
	for _, deployment := range deploymentMap {
		deployments = append(deployments, deployment)
	}

	// Sort by creation date (most recent first)
	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].CreatedAt.After(deployments[j].CreatedAt)
	})

	return deployments, nil
}

func (s *deploymentService) parseEventLine(line, appName string) *application.Deployment {
	// Typical format: "Jul  3 16:10:03 dokku.me dokku[128195]: INVOKED: pre-deploy( pythonapp )"
	// or: "Jul  3 16:10:24 dokku.me dokku[129451]: INVOKED: check-deploy( pythonapp 6274ced0d4be11af4490cd18abaf77cdd593f025133f403d984e80d86a39acec web 5000 10.0.16.80 )"

	parts := strings.Fields(line)
	if len(parts) < 6 {
		return nil
	}

	// Extract the date (format Jul  3 16:10:03)
	dateStr := strings.Join(parts[0:3], " ")
	timestamp, err := s.parseEventTimestamp(dateStr)
	if err != nil {
		s.logger.Debug("Failed to analyze event date",
			"error", err)
		timestamp = time.Now()
	}

	// Search for deployment events
	eventContent := strings.Join(parts[4:], " ")

	if strings.Contains(eventContent, "pre-deploy(") {
		return s.createDeploymentFromPreDeploy(appName, timestamp, eventContent)
	} else if strings.Contains(eventContent, "check-deploy(") {
		return s.createDeploymentFromCheckDeploy(appName, timestamp, eventContent)
	} else if strings.Contains(eventContent, "post-deploy(") {
		return s.updateDeploymentFromPostDeploy(appName, timestamp, eventContent)
	}

	return nil
}

type deploymentBuilder struct {
	appName   string
	timestamp time.Time
	gitRef    string
	status    application.DeploymentStatus
}

func newDeploymentBuilder(appName string, timestamp time.Time) *deploymentBuilder {
	return &deploymentBuilder{
		appName:   appName,
		timestamp: timestamp,
		gitRef:    "unknown",
		status:    application.DeploymentStatusRunning,
	}
}

func (db *deploymentBuilder) withGitRef(gitRef string) *deploymentBuilder {
	db.gitRef = gitRef
	return db
}

func (db *deploymentBuilder) withStatus(status application.DeploymentStatus) *deploymentBuilder {
	db.status = status
	return db
}

func (db *deploymentBuilder) build() *application.Deployment {
	deploymentID := fmt.Sprintf("deploy_%d", db.timestamp.UnixNano())
	if db.gitRef != "unknown" {
		deploymentID = fmt.Sprintf("deploy_%s_%d", db.gitRef, db.timestamp.UnixNano())
	}

	deployment := &application.Deployment{
		ID:        deploymentID,
		AppName:   db.appName,
		GitRef:    db.gitRef,
		Status:    db.status,
		CreatedAt: db.timestamp,
	}

	switch db.status {
	case application.DeploymentStatusRunning:
		deployment.StartedAt = &db.timestamp
	case application.DeploymentStatusSucceeded:
		deployment.CompletedAt = &db.timestamp
	}

	return deployment
}

func (s *deploymentService) createDeploymentFromPreDeploy(appName string, timestamp time.Time, eventContent string) *application.Deployment {
	return newDeploymentBuilder(appName, timestamp).build()
}

func (s *deploymentService) createDeploymentFromCheckDeploy(appName string, timestamp time.Time, eventContent string) *application.Deployment {
	gitRef := s.extractGitRefFromCheckDeploy(eventContent)
	return newDeploymentBuilder(appName, timestamp).withGitRef(gitRef).build()
}

func (s *deploymentService) updateDeploymentFromPostDeploy(appName string, timestamp time.Time, eventContent string) *application.Deployment {
	return newDeploymentBuilder(appName, timestamp).
		withStatus(application.DeploymentStatusSucceeded).
		build()
}

func (s *deploymentService) extractGitRefFromCheckDeploy(eventContent string) string {
	// Format: "check-deploy( pythonapp 6274ced0d4be11af4490cd18abaf77cdd593f025133f403d984e80d86a39acec web 5000 10.0.16.80 )"

	start := strings.Index(eventContent, "(")
	if start == -1 {
		return "unknown"
	}

	end := strings.Index(eventContent[start:], ")")
	if end == -1 {
		return "unknown"
	}

	params := strings.Fields(eventContent[start+1 : start+end])
	if len(params) < 2 {
		return "unknown"
	}

	possibleSHA := params[1]
	if len(possibleSHA) >= 8 && len(possibleSHA) <= 64 {
		return possibleSHA[:8] // 8 first characters of the SHA
	}

	return "unknown"
}

func (s *deploymentService) parseEventTimestamp(dateStr string) (time.Time, error) {
	// Typical format: "Jul  3 16:10:03"
	// Add the current year because it is not in the logs
	currentYear := time.Now().Year()
	fullDateStr := fmt.Sprintf("%s %d", dateStr, currentYear)

	// Parse with the appropriate format
	return time.Parse("Jan  2 15:04:05 2006", fullDateStr)
}

// Récupère le déploiement actuel comme fallback
func (s *deploymentService) getCurrentDeploymentAsFallback(ctx context.Context, appName string) ([]*application.Deployment, error) {
	s.logger.Debug("Using fallback to retrieve the current deployment",
		"app_name", appName)

	// Use git:report to obtain the current SHA
	output, err := s.client.ExecuteCommand(ctx, "git:report", []string{appName})
	if err != nil {
		s.logger.Debug("Failed to retrieve git:report",
			"error", err)
		return []*application.Deployment{}, nil
	}

	gitRef := s.parseGitRefFromReport(string(output))
	if gitRef == "" {
		gitRef = "unknown"
	}

	// Create a basic deployment with the available information
	deployment := &application.Deployment{
		ID:        fmt.Sprintf("current_%s", gitRef),
		AppName:   appName,
		GitRef:    gitRef,
		Status:    application.DeploymentStatusSucceeded,
		CreatedAt: time.Now(),
	}

	return []*application.Deployment{deployment}, nil
}

// Extracts the Git SHA from the git:report output
func (s *deploymentService) parseGitRefFromReport(reportOutput string) string {
	lines := strings.Split(reportOutput, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Search for a line containing "Git sha" or "Source sha"
		if strings.Contains(strings.ToLower(line), "sha") && strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				sha := strings.TrimSpace(parts[1])
				if len(sha) >= 8 {
					return sha[:8]
				}
			}
		}
	}
	return ""
}

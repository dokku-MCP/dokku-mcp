package dokkutesting

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/alex-galey/dokku-mcp/src/domain/application"
)

// MockDokkuClient implements DokkuClient for tests without real server
type MockDokkuClient struct {
	logger      *slog.Logger
	apps        map[string]*MockApp
	commandLogs []MockCommand
}

type MockApp struct {
	Name   string
	Config map[string]string
	Info   map[string]interface{}
}

type MockCommand struct {
	Command string
	Args    []string
	Result  []byte
	Error   error
}

// NewMockDokkuClient creates a new mock client
func NewMockDokkuClient(logger *slog.Logger) *MockDokkuClient {
	return &MockDokkuClient{
		logger:      logger,
		apps:        make(map[string]*MockApp),
		commandLogs: make([]MockCommand, 0),
	}
}

// validateAppName validates that an application name conforms to Dokku rules
func validateAppName(name string) error {
	if name == "" {
		return fmt.Errorf("application name required")
	}

	// Dokku doesn't accept slashes in application names
	if strings.Contains(name, "/") {
		return fmt.Errorf("invalid application name: '/' characters are not allowed")
	}

	// Dokku doesn't accept special characters
	if strings.ContainsAny(name, " @#$%^&*()+=[]{}|\\:;\"'<>?") {
		return fmt.Errorf("invalid application name: special characters not allowed")
	}

	return nil
}

// ExecuteCommand simule l'exécution d'une commande Dokku
func (m *MockDokkuClient) ExecuteCommand(ctx context.Context, command string, args []string) ([]byte, error) {
	m.logger.Debug("Mock command executed",
		"command", command, "args", args)

	cmd := MockCommand{
		Command: command,
		Args:    args,
	}

	// Utilise une méthode spécialisée selon la commande
	switch command {
	case "apps:create":
		cmd.Result, cmd.Error = m.handleAppsCreate(args)
	case "apps:exists":
		cmd.Result, cmd.Error = m.handleAppsExists(args)
	case "apps:destroy":
		cmd.Result, cmd.Error = m.handleAppsDestroy(args)
	case "apps:list":
		cmd.Result, cmd.Error = m.handleAppsList()
	case "events":
		cmd.Result, cmd.Error = m.handleEvents()
	case "git:report":
		cmd.Result, cmd.Error = m.handleGitReport(args)
	case "config:show":
		cmd.Result, cmd.Error = m.handleConfigShow(args)
	case "config:set":
		cmd.Result, cmd.Error = m.handleConfigSet(args)
	default:
		cmd.Error = fmt.Errorf("mock command not implemented: %s", command)
	}

	m.commandLogs = append(m.commandLogs, cmd)
	return cmd.Result, cmd.Error
}

// handleAppsCreate traite la commande apps:create
func (m *MockDokkuClient) handleAppsCreate(args []string) ([]byte, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("application name required")
	}

	appName := args[0]
	if err := validateAppName(appName); err != nil {
		return nil, err
	}

	if _, exists := m.apps[appName]; exists {
		return nil, fmt.Errorf("application %s already exists", appName)
	}

	m.apps[appName] = &MockApp{
		Name:   appName,
		Config: make(map[string]string),
		Info: map[string]interface{}{
			"deployed": false,
			"running":  false,
		},
	}

	return []byte(fmt.Sprintf("Application %s created", appName)), nil
}

// handleAppsExists traite la commande apps:exists
func (m *MockDokkuClient) handleAppsExists(args []string) ([]byte, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("application name required")
	}

	appName := args[0]
	if _, exists := m.apps[appName]; !exists {
		return nil, fmt.Errorf("application %s does not exist", appName)
	}

	return []byte("Application exists"), nil
}

// handleAppsDestroy traite la commande apps:destroy
func (m *MockDokkuClient) handleAppsDestroy(args []string) ([]byte, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("application name required")
	}

	appName := args[0]
	if _, exists := m.apps[appName]; !exists {
		return nil, fmt.Errorf("application %s does not exist", appName)
	}

	delete(m.apps, appName)
	return []byte(fmt.Sprintf("Application %s deleted", appName)), nil
}

// handleAppsList traite la commande apps:list
func (m *MockDokkuClient) handleAppsList() ([]byte, error) {
	appList := make([]string, 0, len(m.apps))
	for name := range m.apps {
		appList = append(appList, name)
	}

	result := ""
	for _, name := range appList {
		result += name + "\n"
	}

	return []byte(result), nil
}

// handleEvents traite la commande events
func (m *MockDokkuClient) handleEvents() ([]byte, error) {
	// Simule des événements vides
	return []byte(""), nil
}

// handleGitReport traite la commande git:report
func (m *MockDokkuClient) handleGitReport(args []string) ([]byte, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("application name required")
	}

	appName := args[0]
	if _, exists := m.apps[appName]; !exists {
		return nil, fmt.Errorf("application %s does not exist", appName)
	}

	result := fmt.Sprintf("%s git information:\nGit sha: abc123\nDeploy source: git", appName)
	return []byte(result), nil
}

// handleConfigShow traite la commande config:show
func (m *MockDokkuClient) handleConfigShow(args []string) ([]byte, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("application name required")
	}

	appName := args[0]
	app, exists := m.apps[appName]
	if !exists {
		return nil, fmt.Errorf("application %s does not exist", appName)
	}

	result := ""
	for key, value := range app.Config {
		result += fmt.Sprintf("%s=%s\n", key, value)
	}

	return []byte(result), nil
}

// handleConfigSet traite la commande config:set
func (m *MockDokkuClient) handleConfigSet(args []string) ([]byte, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("application name and configuration required")
	}

	appName := args[0]
	app, exists := m.apps[appName]
	if !exists {
		return nil, fmt.Errorf("application %s does not exist", appName)
	}

	// Parse les paires key=value
	for _, arg := range args[1:] {
		if len(arg) > 0 && arg != "--no-restart" {
			key, value := m.parseKeyValue(arg)
			if key != "" {
				app.Config[key] = value
			}
		}
	}

	return []byte("Configuration updated"), nil
}

// parseKeyValue analyse une chaîne key=value
func (m *MockDokkuClient) parseKeyValue(arg string) (string, string) {
	parts := []rune(arg)
	for i, char := range parts {
		if char == '=' {
			return string(parts[:i]), string(parts[i+1:])
		}
	}
	return "", ""
}

func (m *MockDokkuClient) GetApplications(ctx context.Context) ([]string, error) {
	apps := make([]string, 0, len(m.apps))
	for name := range m.apps {
		apps = append(apps, name)
	}

	return apps, nil
}

func (m *MockDokkuClient) GetApplicationInfo(ctx context.Context, appName string) (map[string]interface{}, error) {
	if app, exists := m.apps[appName]; exists {
		return app.Info, nil
	}
	return nil, fmt.Errorf("application %s does not exist", appName)
}

func (m *MockDokkuClient) GetApplicationConfig(ctx context.Context, appName string) (map[string]string, error) {
	if app, exists := m.apps[appName]; exists {
		return app.Config, nil
	}
	return nil, fmt.Errorf("application %s does not exist", appName)
}

func (m *MockDokkuClient) SetApplicationConfig(ctx context.Context, appName string, config map[string]string) error {
	if app, exists := m.apps[appName]; exists {
		for key, value := range config {
			app.Config[key] = value
		}
		return nil
	}
	return fmt.Errorf("application %s does not exist", appName)
}

func (m *MockDokkuClient) GetCommandLogs() []MockCommand {
	return m.commandLogs
}

// MockDeploymentService implements DeploymentService with fake data
type MockDeploymentService struct {
	logger      *slog.Logger
	deployments map[string][]*application.Deployment
}

func NewMockDeploymentService(logger *slog.Logger) *MockDeploymentService {
	return &MockDeploymentService{
		logger:      logger,
		deployments: make(map[string][]*application.Deployment),
	}
}

// Deploy simulates a deployment
func (m *MockDeploymentService) Deploy(ctx context.Context, appName string, options application.DeployOptions) (*application.Deployment, error) {
	deployment, err := application.NewDeployment(appName, options.GitRef)
	if err != nil {
		return nil, err
	}

	// Simulate successful deployment
	deployment.Start()
	time.Sleep(100 * time.Millisecond) // Simulate processing time
	deployment.Complete()

	if m.deployments[appName] == nil {
		m.deployments[appName] = make([]*application.Deployment, 0)
	}
	m.deployments[appName] = append(m.deployments[appName], deployment)

	m.logger.Info("Mock deployment completed", "app", appName, "deploy_id", deployment.ID)
	return deployment, nil
}

// Rollback simulates a rollback
func (m *MockDeploymentService) Rollback(ctx context.Context, appName string, version string) error {
	if deployments, exists := m.deployments[appName]; exists && len(deployments) > 0 {
		// Mark the last deployment as rolled back
		deployments[len(deployments)-1].Rollback()
		m.logger.Info("Mock rollback completed", "app", appName, "version", version)
		return nil
	}
	return fmt.Errorf("no deployments found for application %s", appName)
}

func (m *MockDeploymentService) GetHistory(ctx context.Context, appName string) ([]*application.Deployment, error) {
	if deployments, exists := m.deployments[appName]; exists {
		return deployments, nil
	}
	// Return empty list for applications without deployments
	return make([]*application.Deployment, 0), nil
}

type MockClientFactory struct{}

func (f *MockClientFactory) CreateClient(config *ClientConfig, logger *slog.Logger) DokkuClient {
	return NewMockDokkuClient(logger)
}

type MockDeploymentServiceFactory struct{}

func (f *MockDeploymentServiceFactory) CreateService(client DokkuClient, logger *slog.Logger) application.DeploymentService {
	return NewMockDeploymentService(logger)
}

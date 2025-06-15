package application

import (
	"fmt"
	"time"
)

type ApplicationState string

const (
	StateCreated  ApplicationState = "created"
	StateDeployed ApplicationState = "deployed"
	StateRunning  ApplicationState = "running"
	StateStopped  ApplicationState = "stopped"
	StateError    ApplicationState = "error"
)

func (s ApplicationState) String() string {
	return string(s)
}

func (s ApplicationState) IsValid() bool {
	switch s {
	case StateCreated, StateDeployed, StateRunning, StateStopped, StateError:
		return true
	default:
		return false
	}
}

type ProcessType string

const (
	ProcessTypeWeb     ProcessType = "web"
	ProcessTypeWorker  ProcessType = "worker"
	ProcessTypeCron    ProcessType = "cron"
	ProcessTypeRelease ProcessType = "release"
)

type Process struct {
	Type    ProcessType `json:"type"`
	Command string      `json:"command"`
	Scale   int         `json:"scale"`
}

func NewProcess(processType ProcessType, command string, scale int) (*Process, error) {
	if command == "" {
		return nil, fmt.Errorf("process command cannot be empty")
	}
	if scale < 0 {
		return nil, fmt.Errorf("process scale cannot be negative")
	}

	return &Process{
		Type:    processType,
		Command: command,
		Scale:   scale,
	}, nil
}

type ApplicationConfig struct {
	BuildPack          *BuildpackName           `json:"buildpack,omitempty"`
	Domains            []*Domain                `json:"domains"`
	EnvironmentVars    map[string]string        `json:"environment_vars"`
	Processes          map[ProcessType]*Process `json:"processes"`
	ResourceLimits     *ResourceLimits          `json:"resource_limits,omitempty"`
	HealthCheckPath    string                   `json:"health_check_path,omitempty"`
	HealthCheckTimeout *time.Duration           `json:"health_check_timeout,omitempty"`
}

type ResourceLimits struct {
	Memory string `json:"memory,omitempty"` // e.g., "512M", "1G"
	CPU    string `json:"cpu,omitempty"`    // e.g., "0.5", "1"
}

func NewApplicationConfig() *ApplicationConfig {
	return &ApplicationConfig{
		Domains:         make([]*Domain, 0),
		EnvironmentVars: make(map[string]string),
		Processes:       make(map[ProcessType]*Process),
	}
}

func (c *ApplicationConfig) AddDomain(domain *Domain) error {
	if domain == nil {
		return fmt.Errorf("le domaine ne peut pas être nul")
	}

	// Vérifier les doublons
	for _, existingDomain := range c.Domains {
		if existingDomain.Equal(domain) {
			return fmt.Errorf("le domaine %s existe déjà", domain.Value())
		}
	}

	c.Domains = append(c.Domains, domain)
	return nil
}

// AddDomainString ajoute un domaine à partir d'une chaîne (helper method)
func (c *ApplicationConfig) AddDomainString(domainStr string) error {
	domain, err := NewDomain(domainStr)
	if err != nil {
		return fmt.Errorf("domaine invalide: %w", err)
	}
	return c.AddDomain(domain)
}

// RemoveDomain supprime un domaine
func (c *ApplicationConfig) RemoveDomain(domain *Domain) error {
	if domain == nil {
		return fmt.Errorf("le domaine ne peut pas être nul")
	}

	for i, existingDomain := range c.Domains {
		if existingDomain.Equal(domain) {
			c.Domains = append(c.Domains[:i], c.Domains[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("le domaine %s n'existe pas", domain.Value())
}

// GetDomainStrings retourne les domaines sous forme de chaînes (helper method)
func (c *ApplicationConfig) GetDomainStrings() []string {
	domains := make([]string, len(c.Domains))
	for i, domain := range c.Domains {
		domains[i] = domain.Value()
	}
	return domains
}

func (c *ApplicationConfig) SetEnvironmentVar(key, value string) error {
	if key == "" {
		return fmt.Errorf("environment variable key cannot be empty")
	}

	c.EnvironmentVars[key] = value
	return nil
}

func (c *ApplicationConfig) AddProcess(process *Process) error {
	if process == nil {
		return fmt.Errorf("process cannot be nil")
	}

	c.Processes[process.Type] = process
	return nil
}

// SetBuildpack définit le buildpack de l'application
func (c *ApplicationConfig) SetBuildpack(buildpack *BuildpackName) {
	c.BuildPack = buildpack
}

// SetBuildpackString définit le buildpack à partir d'une chaîne (helper method)
func (c *ApplicationConfig) SetBuildpackString(buildpackStr string) error {
	if buildpackStr == "" {
		c.BuildPack = nil
		return nil
	}

	buildpack, err := NewBuildpackName(buildpackStr)
	if err != nil {
		return fmt.Errorf("buildpack invalide: %w", err)
	}

	c.BuildPack = buildpack
	return nil
}

type Application struct {
	name       string
	state      ApplicationState
	config     *ApplicationConfig
	createdAt  time.Time
	updatedAt  time.Time
	lastDeploy *time.Time
	gitRef     *GitRef
	buildImage string
	runImage   string
}

func NewApplication(name string) (*Application, error) {
	if name == "" {
		return nil, fmt.Errorf("application name cannot be empty")
	}

	if len(name) > 63 { // DNS label limit
		return nil, fmt.Errorf("application name cannot exceed 63 characters")
	}

	now := time.Now()
	return &Application{
		name:      name,
		state:     StateCreated,
		config:    NewApplicationConfig(),
		createdAt: now,
		updatedAt: now,
	}, nil
}

func (a *Application) Name() string               { return a.name }
func (a *Application) State() ApplicationState    { return a.state }
func (a *Application) Config() *ApplicationConfig { return a.config }
func (a *Application) CreatedAt() time.Time       { return a.createdAt }
func (a *Application) UpdatedAt() time.Time       { return a.updatedAt }
func (a *Application) LastDeploy() *time.Time     { return a.lastDeploy }
func (a *Application) GitRef() *GitRef            { return a.gitRef }
func (a *Application) BuildImage() string         { return a.buildImage }
func (a *Application) RunImage() string           { return a.runImage }

// GitRefString retourne la référence Git sous forme de chaîne (helper method)
func (a *Application) GitRefString() string {
	if a.gitRef == nil {
		return ""
	}
	return a.gitRef.Value()
}

func (a *Application) UpdateState(newState ApplicationState) error {
	if !newState.IsValid() {
		return fmt.Errorf("invalid application state: %s", newState)
	}

	if !a.isValidStateTransition(a.state, newState) {
		return fmt.Errorf("invalid state transition from %s to %s", a.state, newState)
	}

	a.state = newState
	a.updatedAt = time.Now()

	return nil
}

// Deploy effectue un déploiement avec les nouveaux types
func (a *Application) Deploy(gitRef *GitRef, buildImage, runImage string) error {
	if gitRef == nil {
		return fmt.Errorf("la référence Git ne peut pas être nulle")
	}

	a.gitRef = gitRef
	a.buildImage = buildImage
	a.runImage = runImage

	now := time.Now()
	a.lastDeploy = &now
	a.updatedAt = now

	return a.UpdateState(StateDeployed)
}

// DeployString effectue un déploiement à partir d'une chaîne Git (helper method)
func (a *Application) DeployString(gitRefStr, buildImage, runImage string) error {
	gitRef, err := NewGitRef(gitRefStr)
	if err != nil {
		return fmt.Errorf("référence Git invalide: %w", err)
	}

	return a.Deploy(gitRef, buildImage, runImage)
}

func (a *Application) UpdateConfig(config *ApplicationConfig) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}

	a.config = config
	a.updatedAt = time.Now()

	return nil
}

func (a *Application) IsRunning() bool {
	return a.state == StateRunning
}

func (a *Application) IsDeployed() bool {
	return a.state == StateDeployed || a.state == StateRunning
}

// HasDomain vérifie si l'application a un domaine spécifique
func (a *Application) HasDomain(domain *Domain) bool {
	if domain == nil {
		return false
	}

	for _, d := range a.config.Domains {
		if d.Equal(domain) {
			return true
		}
	}
	return false
}

// HasDomainString vérifie si l'application a un domaine spécifique (helper method)
func (a *Application) HasDomainString(domainStr string) bool {
	domain, err := NewDomain(domainStr)
	if err != nil {
		return false
	}
	return a.HasDomain(domain)
}

func (a *Application) GetProcessScale(processType ProcessType) int {
	if process, exists := a.config.Processes[processType]; exists {
		return process.Scale
	}
	return 0
}

// GetBuildpack retourne le buildpack de l'application
func (a *Application) GetBuildpack() *BuildpackName {
	return a.config.BuildPack
}

// GetBuildpackString retourne le buildpack sous forme de chaîne (helper method)
func (a *Application) GetBuildpackString() string {
	if a.config.BuildPack == nil {
		return ""
	}
	return a.config.BuildPack.Value()
}

func (a *Application) isValidStateTransition(from, to ApplicationState) bool {
	validTransitions := map[ApplicationState][]ApplicationState{
		StateCreated:  {StateDeployed, StateError},
		StateDeployed: {StateRunning, StateStopped, StateError},
		StateRunning:  {StateStopped, StateError, StateDeployed},
		StateStopped:  {StateRunning, StateError, StateDeployed},
		StateError:    {StateCreated, StateDeployed, StateRunning, StateStopped},
	}

	validToStates, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, validTo := range validToStates {
		if validTo == to {
			return true
		}
	}

	return false
}
